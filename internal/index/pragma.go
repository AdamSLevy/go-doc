package index

import (
	"context"
	"fmt"
)

// sqliteApplicationID is the magic number used to identify sqlite3 databases
// created by this application.
//
// See https://www.sqlite.org/fileformat.html#application_id
const (
	sqliteApplicationID int32 = 0x0_90_D0C_90 // GO DOC GO
	pragmaApplicationID       = "application_id"
)

func (idx *Index) checkSetApplicationID(ctx context.Context) error {
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
	const pragmaSchemaVersion = "schema_version"
	var schemaVersion int
	if err := idx.readPragma(ctx, pragmaSchemaVersion, &schemaVersion); err != nil {
		return 0, err
	}
	return schemaVersion, nil
}

func (idx *Index) enableForeignKeys(ctx context.Context) error {
	const pragmaForeignKeys = "foreign_keys"
	return idx.setPragma(ctx, pragmaForeignKeys, "on")
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
