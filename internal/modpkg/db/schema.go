package db

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"hash/crc32"

	_ "modernc.org/sqlite"
)

// applySchema execs all schemaQueries against the db.
func applySchema(ctx context.Context, db Querier) error {
	if err := execQueries(ctx, db, schemaQueries...); err != nil {
		return err
	}

	if err := setApplicationID(ctx, db); err != nil {
		return err
	}

	return setUserVersion(ctx, db, schemaChecksum)
}

// schemaChecksum is the CRC32 checksum of schema.
var schemaChecksum int32 = func() int32 {
	crc := crc32.NewIEEE()
	for _, stmt := range schemaQueries {
		if _, err := crc.Write([]byte(stmt)); err != nil {
			panic(err)
		}
	}
	return int32(crc.Sum32())
}()

// rawSchema is the SQL rawSchema for the index database.
//
//go:embed schema.sql
var rawSchema []byte

var schemaQueries = func() []string {
	queries, err := splitSQL(rawSchema)
	if err != nil {
		panic(err)
	}
	return queries
}()

func execQueries(ctx context.Context, db Querier, queries ...string) error {
	for _, query := range queries {
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to apply query: %w\n%s\n", err, query)
		}
	}
	return nil
}
func splitSQL(sql []byte) (queries []string, _ error) {
	scanner := bufio.NewScanner(bytes.NewReader(sql))
	scanner.Split(sqlSplit)
	for scanner.Scan() {
		query := scanner.Text()
		if query == "" {
			continue
		}
		queries = append(queries, query)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to split SQL statements: %w", err)
	}
	return queries, nil
}

const stmtDelimiter = ";---"

func sqlSplit(data []byte, atEOF bool) (advance int, token []byte, rerr error) {
	defer func() {
		if (rerr != nil &&
			!errors.Is(rerr, bufio.ErrFinalToken)) ||
			advance == 0 {
			return
		}
		// Trim the token of any leading or trailing whitespace.
		token = bytes.TrimSpace(token)
		// Trim leading comment lines.
		const commentPrefix = "--"
		for {
			adv, tkn, err := bufio.ScanLines(token, true)
			if err != nil {
				rerr = err
				return
			}
			if adv == 0 {
				return
			}
			if len(tkn) > 0 {
				tkn = bytes.TrimSpace(tkn)
				if isComment := bytes.HasPrefix(tkn, []byte(commentPrefix)); !isComment {
					return
				}
			}
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
			return len(data), data, bufio.ErrFinalToken
		}
		// Ask for more data so we can find the EOL.
		return 0, nil, nil
	}
	// We found the stmtDelimiter, now find the next newline.
	newline := bytes.Index(data[stmtDelim+len([]byte(stmtDelimiter)):], []byte("\n"))
	if newline == -1 {
		if atEOF {
			return len(data), data, bufio.ErrFinalToken
		}
		return 0, nil, nil
	}

	return stmtDelim + len([]byte(stmtDelimiter)) + newline + 1, data[:stmtDelim+1], nil
}
