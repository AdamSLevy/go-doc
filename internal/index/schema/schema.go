package schema

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"hash/crc32"

	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/dlog"
)

func OpenDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := initialize(ctx, db); err != nil {
		if err := db.Close(); err != nil {
			dlog.Printf("failed to close database: %v", err)
		}
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	return db, nil
}

func initialize(ctx context.Context, db *sql.DB) (rerr error) {
	// Only proceed if the database matches our application ID.
	if err := assertApplicationID(ctx, db); err != nil {
		return err
	}

	// Check if the schema is up to date.
	userVersion, err := getUserVersion(ctx, db)
	if err != nil {
		return err
	}
	if userVersion == schemaCRC {
		// Schema is up to date.
		return nil
	}
	if userVersion != 0 {
		return fmt.Errorf("database has incorrect user_version")
	}
	dlog.Printf("Applying schema...")

	schemaVersion, err := getSchemaVersion(ctx, db)
	if err != nil {
		return err
	}
	if schemaVersion > 0 {
		return fmt.Errorf("database schema_version (%d) is not zero", schemaVersion)
	}

	if err := enableForeignKeys(ctx, db); err != nil {
		return err
	}

	if err := enableRecursiveTriggers(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer CommitOrRollback(tx, &rerr)

	return applySchema(ctx, db)
}

func CommitOrRollback(tx *sql.Tx, rerr *error) {
	if *rerr != nil {
		if err := tx.Rollback(); err != nil {
			*rerr = errors.Join(*rerr, fmt.Errorf("failed to rollback transaction: %w", err))
		}
		return
	}
	if err := tx.Commit(); err != nil {
		*rerr = fmt.Errorf("failed to commit transaction: %w", err)
	}
}

func applySchema(ctx context.Context, db Querier) error {
	_, err := db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	dlog.Printf("schema CRC: %d", schemaCRC)
	return setUserVersion(ctx, db, schemaCRC)
}

// schema is the SQL schema for the index database.
//
//go:embed schema.sql
var schema string

// schemaCRC is the CRC32 checksum of schema.
var schemaCRC = func() uint32 {
	crc := crc32.NewIEEE()
	crc.Write([]byte(schema))
	return crc.Sum32()
}()
