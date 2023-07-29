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
	sqliteApplicationID uint32 = 0x0_90_D0C_90 // GO DOC GO
	pragmaApplicationID        = "application_id"
)

func (idx *Index) assertApplicationID(ctx context.Context) error {
	appID, err := idx.getApplicationID(ctx)
	if err != nil {
		return err
	}
	if appID == 0 { // app ID not set
		return idx.setApplicationID(ctx)
	}
	if appID != sqliteApplicationID {
		return fmt.Errorf("unrecognized database")
	}
	return nil
}
func (idx *Index) getApplicationID(ctx context.Context) (appID uint32, _ error) {
	return appID, idx.getPragma(ctx, pragmaApplicationID, &appID)
}
func (idx *Index) setApplicationID(ctx context.Context) error {
	return idx.setPragma(ctx, pragmaApplicationID, sqliteApplicationID)
}

const pragmaUserVersion = "user_version"

func (idx *Index) getUserVersion(ctx context.Context) (userVersion uint32, _ error) {
	return userVersion, idx.getPragma(ctx, pragmaUserVersion, &userVersion)
}
func (idx *Index) setUserVersion(ctx context.Context, userVersion uint32) error {
	return idx.setPragma(ctx, pragmaUserVersion, userVersion)
}

func (idx *Index) getSchemaVersion(ctx context.Context) (int, error) {
	const pragmaSchemaVersion = "schema_version"
	var schemaVersion int
	if err := idx.getPragma(ctx, pragmaSchemaVersion, &schemaVersion); err != nil {
		return 0, err
	}
	return schemaVersion, nil
}

func (idx *Index) enableForeignKeys(ctx context.Context) error {
	const pragmaForeignKeys = "foreign_keys"
	return idx.setPragma(ctx, pragmaForeignKeys, "on")
}

func (idx *Index) getPragma(ctx context.Context, key string, val any) error {
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
