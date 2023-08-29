package schema

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
func getApplicationID(ctx context.Context, db Querier) (appID uint32, _ error) {
	return appID, getPragma(ctx, db, pragmaApplicationID, &appID)
}
func setApplicationID(ctx context.Context, db Querier) error {
	return setPragma(ctx, db, pragmaApplicationID, sqliteApplicationID)
}

const pragmaUserVersion = "user_version"

func getUserVersion(ctx context.Context, db Querier) (userVersion uint32, _ error) {
	return userVersion, getPragma(ctx, db, pragmaUserVersion, &userVersion)
}
func setUserVersion(ctx context.Context, db Querier, userVersion uint32) error {
	return setPragma(ctx, db, pragmaUserVersion, userVersion)
}

func getSchemaVersion(ctx context.Context, db Querier) (int, error) {
	const pragmaSchemaVersion = "schema_version"
	var schemaVersion int
	if err := getPragma(ctx, db, pragmaSchemaVersion, &schemaVersion); err != nil {
		return 0, err
	}
	return schemaVersion, nil
}

func enableForeignKeys(ctx context.Context, db Querier) error {
	const pragmaForeignKeys = "foreign_keys"
	return setPragma(ctx, db, pragmaForeignKeys, "on")
}

func enableRecursiveTriggers(ctx context.Context, db Querier) error {
	const pragmaRecursiveTriggers = "recursive_triggers"
	return setPragma(ctx, db, pragmaRecursiveTriggers, "on")
}

func getPragma(ctx context.Context, db Querier, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s;`, key)
	err := db.QueryRowContext(ctx, query).Scan(val)
	if err != nil {
		return fmt.Errorf("failed to read pragma %s: %w", key, err)
	}
	return nil
}

func setPragma(ctx context.Context, db Querier, key string, val any) error {
	query := fmt.Sprintf(`PRAGMA %s=%v;`, key, val)
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to set pragma %s=%v: %w", key, val, err)
	}
	return nil
}
