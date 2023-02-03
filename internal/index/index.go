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

// sqliteApplicationID is the magic number used to identify sqlite3 databases
// created by this application.
//
// See https://www.sqlite.org/fileformat.html#application_id
const sqliteApplicationID int32 = 0x0_90_D0C_90 // GO DOC GO

type Index struct {
	options

	db *sql.DB
	tx *sql.Tx

	preparedStmt preparedStmt

	sync
	cancel context.CancelFunc
	g      *errgroup.Group
}

type preparedStmt struct {
	loadModule   *sql.Stmt
	insertModule *sql.Stmt
	updateModule *sql.Stmt

	getPackageID  *sql.Stmt
	insertPackage *sql.Stmt

	insertPartial *sql.Stmt
}

func Load(ctx context.Context, dbPath string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil, nil
	}

	dlog.Printf("loading %q", dbPath)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	idx := Index{
		options: o,
		db:      db,
	}

	if err := idx.checkSetApplicationID(ctx); err != nil {
		return nil, err
	}

	ctx, idx.cancel = context.WithCancel(ctx)
	idx.g, ctx = errgroup.WithContext(ctx)
	idx.g.Go(func() error {
		defer idx.cancel()
		return idx.initSync(ctx, codeRoots)
	})

	return &idx, nil
}

func (idx *Index) Close() error {
	idx.cancel()
	if err := idx.waitSync(); err != nil {
		dlog.Printf("failed to sync: %v", err)
	}
	return idx.db.Close()
}
func (idx *Index) waitSync() error { return idx.g.Wait() }

func (idx *Index) initSync(ctx context.Context, codeRoots []godoc.PackageDir) error {
	if err := idx.enableForeignKeys(ctx); err != nil {
		return err
	}

	if err := idx.applySchema(ctx); err != nil {
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

func (idx *Index) applySchema(ctx context.Context) error {
	schemaVersion, err := idx.schemaVersion(ctx)
	if err != nil {
		return err
	}

	schema := schema()
	if schemaVersion > len(schema) {
		return fmt.Errorf("database schema version (%d) higher than supported (<=%d)", schemaVersion, len(schema))
	}
	dlog.Printf("schema version: %d of %d", schemaVersion, len(schema))
	if schemaVersion == len(schema) {
		return nil
	}
	// Apply all schema updates, which should be idempotent.
	for i, stmt := range schema {
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
