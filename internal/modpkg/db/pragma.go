package db

import (
	"context"
	"fmt"
)

// sqliteApplicationID is the magic number used to identify sqlite3 databases
// created by this application.
//
// See https://www.sqlite.org/fileformat.html#application_id
const (
	sqliteApplicationID     = int32(0x0_90_D0C_90) // GO DOC GO
	pragmaApplicationID     = "application_id"
	pragmaUserVersion       = "user_version"
	pragmaSchemaVersion     = "schema_version"
	pragmaForeignKeys       = "foreign_keys"
	pragmaRecursiveTriggers = "recursive_triggers"
	pragmaJournalMode       = "journal_mode"
)

func assertApplicationID(ctx context.Context, db Querier) error {
	appID, err := getApplicationID(ctx, db)
	if err != nil {
		return err
	}
	if appID == 0 { // app ID not set
		return setApplicationID(ctx, db)
	}
	if appID != sqliteApplicationID {
		return fmt.Errorf("unrecognized database application ID")
	}
	return nil
}
func getApplicationID(ctx context.Context, db Querier) (appID int32, err error) {
	err = getPragma(ctx, db, pragmaApplicationID, &appID)
	return
}
func setApplicationID(ctx context.Context, db Querier) error {
	return setPragma(ctx, db, pragmaApplicationID, sqliteApplicationID)
}

func getUserVersion(ctx context.Context, db Querier) (userVersion int32, err error) {
	err = getPragma(ctx, db, pragmaUserVersion, &userVersion)
	return
}
func setUserVersion(ctx context.Context, db Querier, userVersion int32) error {
	return setPragma(ctx, db, pragmaUserVersion, userVersion)
}

func getSchemaVersion(ctx context.Context, db Querier) (schemaVersion int32, err error) {
	err = getPragma(ctx, db, pragmaSchemaVersion, &schemaVersion)
	return
}

func (db *DB) enableForeignKeys(ctx context.Context) error {
	return setPragma(ctx, db.db, pragmaForeignKeys, true)
}

func (db *DB) enableRecursiveTriggers(ctx context.Context) error {
	return setPragma(ctx, db.db, pragmaRecursiveTriggers, true)
}

func (db *DB) journalModeWAL(ctx context.Context) error {
	return setPragma(ctx, db.db, pragmaJournalMode, "wal")
}

func getPragma(ctx context.Context, db Querier, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s;`, key)
	row := db.QueryRowContext(ctx, query)
	if err := row.Err(); err != nil {
		return fmt.Errorf("failed to query %s: %w", query, err)
	}
	if err := row.Scan(val); err != nil {
		return fmt.Errorf("failed to scan %s: %w", query, err)
	}
	return nil
}

func setPragma(ctx context.Context, db Querier, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s=%v;`, key, val)
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to set %s: %w", query, err)
	}
	return nil
}
