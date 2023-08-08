package schema

import (
	"bufio"
	"bytes"
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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer commitOrRollback(tx, &rerr)

	return applySchema(ctx, db)
}

func commitOrRollback(tx *sql.Tx, rerr *error) {
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
	queries := schemaQueries()
	for i, query := range queries {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to apply schema query %d: %w", i+1, err)
		}
	}

	dlog.Printf("schema CRC: %d", schemaCRC)
	return setUserVersion(ctx, db, schemaCRC)
}

// _schema is the SQL schema for the index database.
//
//go:embed schema.sql
var _schema []byte

var schemaCRC = func() uint32 {
	crc := crc32.NewIEEE()
	crc.Write(_schema)
	return crc.Sum32()
}()

// schemaQueries returns the individual queries in schema.sql.
func schemaQueries() []string {
	const numQueries = 8 // number of queries in schema.sql
	queries := make([]string, 0, numQueries)
	scanner := bufio.NewScanner(bytes.NewReader(_schema))
	scanner.Split(sqlSplit)
	for scanner.Scan() {
		queries = append(queries, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		panic(fmt.Errorf("failed to scan schema.sql: %w", err))
	}
	return queries
}
func sqlSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	defer func() {
		// Trim the token of any leading or trailing whitespace.
		token = bytes.TrimSpace(token)
		if len(token) == 0 {
			// Ensure we don't return an empty token.
			token = nil
		}
	}()

	semiColon := bytes.Index(data, []byte(";"))
	if semiColon == -1 {
		// No semi-colon yet...
		if atEOF {
			// That's everything...
			return len(data), data, nil
		}
		// Ask for more data so we can find the EOL.
		return 0, nil, nil
	}
	// We found a semi-colon...
	return semiColon + 1, data[:semiColon+1], nil
}
