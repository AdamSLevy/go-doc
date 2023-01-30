package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/godoc"
)

const (
	// ApplicationID is the application ID of the database.
	sqliteApplicationID int32 = 0x0_90_D0C_90 // GO DOC GO
)

type Index struct {
	options

	tx *sql.Tx
	db *sql.DB

	_sync
	cancelInitSync context.CancelFunc
	initSyncGroup  *errgroup.Group
}

func Load(ctx context.Context, dbPath string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	idx := Index{
		options: newOptions(opts...),
		db:      db,
	}

	if err := idx.checkSetApplicationID(ctx); err != nil {
		return nil, err
	}

	ctx, idx.cancelInitSync = context.WithCancel(ctx)
	idx.initSyncGroup, ctx = errgroup.WithContext(ctx)
	idx.initSyncGroup.Go(func() error {
		defer idx.cancelInitSync()
		return idx.initSync(ctx, codeRoots)
	})

	return &idx, nil
}

func (idx *Index) Close() error {
	defer idx.db.Close()
	idx.cancelInitSync()
	return idx.waitSync()
}
func (idx *Index) waitSync() error { return idx.initSyncGroup.Wait() }

func (idx *Index) initSync(ctx context.Context, codeRoots []godoc.PackageDir) error {
	if err := idx.enableForeignKeys(ctx); err != nil {
		return err
	}

	if err := idx.updateSchema(ctx); err != nil {
		return err
	}

	if err := idx.loadSync(ctx); ignoreErrNoRows(err) != nil {
		return err
	}

	return idx.syncCodeRoots(ctx, codeRoots)
}
func ignoreErrNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (idx *Index) updateSchema(ctx context.Context) error {
	schemaVersion, err := idx.schemaVersion(ctx)
	if err != nil {
		return err
	}

	schema := schema()
	if schemaVersion > len(schema) {
		return fmt.Errorf("database schema version (%d) higher than supported (<=%d)", schemaVersion, len(schema))
	}
	// Apply any missing schema updates...
	for i, stmt := range schema[schemaVersion:] {
		_, err := idx.db.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to apply schema version %d: %w", i+1, err)
		}
	}
	return nil
}

func (idx *Index) checkSetApplicationID(ctx context.Context) error {
	const pragmaApplicationID = "application_id"
	var appID int32
	if err := idx.readPragma(ctx, pragmaApplicationID, &appID); err != nil {
		return err
	}
	if appID == 0 {
		if err := idx.setPragma(ctx, pragmaApplicationID, sqliteApplicationID); err != nil {
			return err
		}
	} else if appID != sqliteApplicationID {
		return fmt.Errorf("database is not for this application")
	}
	return nil
}

func (idx *Index) schemaVersion(ctx context.Context) (int, error) {
	var schemaVersion int
	if err := idx.readPragma(ctx, "schema_version", &schemaVersion); err != nil {
		return 0, err
	}
	return schemaVersion, nil
}

func (idx *Index) enableForeignKeys(ctx context.Context) error {
	return idx.setPragma(ctx, "foreign_keys", "on")
}

func (idx *Index) readPragma(ctx context.Context, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s;`, key)
	err := idx.db.QueryRowContext(ctx, query).Scan(val)
	if err != nil {
		return fmt.Errorf("failed to read pragma %s: %w", key, err)
	}
	return nil
}

func (idx *Index) setPragma(ctx context.Context, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s=%v;`, key, val)
	_, err := idx.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to set pragma %s=%v: %w", key, val, err)
	}
	return nil
}
