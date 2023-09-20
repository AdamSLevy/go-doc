package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func OpenDB(ctx context.Context, dbPath string) (_ *sql.DB, rerr error) {
	if err := ensureDBPathDir(dbPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() {
		if rerr == nil {
			return
		}
		if err := db.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close database: %w", err))
		}
	}()

	if err := initialize(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}
func ensureDBPathDir(dbPath string) error {
	dirPath := filepath.Dir(dbPath)
	if err := os.Mkdir(dirPath, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create database directory %s: %w", dirPath, err)
	}
	return nil
}

func initialize(ctx context.Context, db *sql.DB) (rerr error) {
	ready, err := checkSchema(ctx, db)
	if err != nil {
		return err
	}

	// Enable foreign keys and recursive triggers.
	if err := enableForeignKeys(ctx, db); err != nil {
		return err
	}
	if err := enableRecursiveTriggers(ctx, db); err != nil {
		return err
	}

	if ready {
		return nil
	}

	return applySchema(ctx, db)
}

func checkSchema(ctx context.Context, db *sql.DB) (ok bool, _ error) {
	appID, err := getApplicationID(ctx, db)
	if err != nil {
		return false, err
	}

	userVersion, err := getUserVersion(ctx, db)
	if err != nil {
		return false, err
	}

	schemaVersion, err := getSchemaVersion(ctx, db)
	if err != nil {
		return false, err
	}

	if appID+
		userVersion+
		schemaVersion == 0 {
		// Database is uninitialized.
		return false, nil
	}

	if appID != sqliteApplicationID {
		return false, fmt.Errorf("unrecognized database application ID")
	}
	if userVersion != schemaChecksum {
		return false, fmt.Errorf("database schema version mismatch")
	}

	return true, nil
}
