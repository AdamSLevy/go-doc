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
		if cerr := db.Close(); cerr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close database: %w", cerr))
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

	// Enable foreign keys and recursive triggers.
	if err := enableForeignKeys(ctx, db); err != nil {
		return err
	}
	if err := enableRecursiveTriggers(ctx, db); err != nil {
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
	dlog.Printf("Applying schema (0x%X)...", uint32(schemaCRC))

	schemaVersion, err := getSchemaVersion(ctx, db)
	if err != nil {
		return err
	}
	if schemaVersion > 0 {
		return fmt.Errorf("database schema_version (%d) is not zero", schemaVersion)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer RollbackOnError(tx, &rerr)
	defer CommitOnSuccess(tx, &rerr)

	return applySchema(ctx, tx)
}

func RollbackOrCommit(tx *sql.Tx, rerr *error) {
	RollbackOnError(tx, rerr)
	CommitOnSuccess(tx, rerr)
}
func RollbackOnError(tx *sql.Tx, rerr *error) {
	if *rerr == nil {
		return
	}
	dlog.Output(0, "rolling back...")
	if err := tx.Rollback(); err != nil {
		*rerr = errors.Join(*rerr, fmt.Errorf("failed to rollback transaction: %w", err))
	}
}
func CommitOnSuccess(tx *sql.Tx, rerr *error) {
	if *rerr != nil {
		return
	}
	if err := tx.Commit(); err != nil {
		*rerr = fmt.Errorf("failed to commit transaction: %w", err)
	}
}

func applySchema(ctx context.Context, db Querier) error {
	if err := execSplit(ctx, db, schema); err != nil {
		return err
	}
	return setUserVersion(ctx, db, schemaCRC)
}

// schema is the SQL schema for the index database.
//
//go:embed schema.sql
var schema []byte

// schemaCRC is the CRC32 checksum of schema.
var schemaCRC int32 = func() int32 {
	crc := crc32.NewIEEE()
	crc.Write(schema)
	return int32(crc.Sum32())
}()

func execSplit(ctx context.Context, db Querier, sql []byte) error {
	return splitSQL(sql, func(query string) error {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to apply query: %w\n%s\n", err, query)
		}
		return nil
	})
}
func splitSQL(sql []byte, handle func(string) error) error {
	scanner := bufio.NewScanner(bytes.NewReader(sql))
	scanner.Split(sqlSplit)
	for scanner.Scan() {
		query := scanner.Text()
		if query == "" {
			continue
		}
		if err := handle(query); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to split SQL statements: %w", err)
	}
	return nil
}

const stmtDelimiter = ";---"

func sqlSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
	defer func() {
		if err != nil || token == nil {
			return
		}
		// Trim the token of any leading or trailing whitespace.
		token = bytes.TrimSpace(token)
		// Trim comment lines.
		const commentPrefix = "--"
		for adv, tkn := 0, []byte(commentPrefix); err == nil &&
			((len(tkn) == 0 && adv > 0) ||
				bytes.HasPrefix(tkn, []byte(commentPrefix))); adv, tkn, err = bufio.ScanLines(token, true) {
			token = token[adv:]
		}
	}()

	stmtDelim := bytes.Index(data, []byte(stmtDelimiter))
	if stmtDelim == -1 {
		// No complete statement yet...
		if atEOF {
			// That's everything... don't treat this as an error to
			// allow for trailing whitespace, comments, or
			// statements that don't use the stmtDelimeter.
			return len(data), data, nil
		}
		// Ask for more data so we can find the EOL.
		return 0, nil, nil
	}
	// We found a semi-colon...
	return stmtDelim + 1, data[:stmtDelim+1], nil
}
