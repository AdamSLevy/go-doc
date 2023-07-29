package index

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/godoc"
)

type Index struct {
	options

	db *sql.DB
	tx *sqlTx

	metadata

	cancel context.CancelFunc
	g      *errgroup.Group
}

func Load(ctx context.Context, dbPath string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil, nil
	}

	dlog.Printf("loading %q", dbPath)
	dlog.Printf("options: %+v", o)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	idx := Index{
		options: o,
		db:      db,
	}

	if err := idx.initDB(ctx); err != nil {
		return nil, err
	}

	ctx, idx.cancel = context.WithCancel(ctx)
	idx.g, ctx = errgroup.WithContext(ctx)
	idx.g.Go(func() error {
		defer idx.cancel()
		return idx.syncCodeRoots(ctx, codeRoots)
	})

	return &idx, nil
}

func (idx *Index) waitSync() error { return idx.g.Wait() }

func (idx *Index) Close() error {
	idx.cancel()
	if err := idx.waitSync(); err != nil {
		dlog.Printf("failed to sync: %v", err)
	}
	return idx.db.Close()
}

func (idx *Index) initDB(ctx context.Context) error {
	// Only proceed if the database matches our application ID.
	if err := idx.assertApplicationID(ctx); err != nil {
		return err
	}

	// Check if the schema is up to date.
	userVersion, err := idx.getUserVersion(ctx)
	if err != nil {
		return err
	}

	if userVersion == 0 { // user version not set
		return idx.applySchema(ctx)
	}

	if userVersion != schemaCRC {
		dlog.Printf("user_version (%d) != schema CRC (%d)", userVersion, schemaCRC)
		return fmt.Errorf("database does not have the correct schema")
	}

	return nil
}

func (idx *Index) applySchema(ctx context.Context) error {
	dlog.Printf("Applying schema...")
	schemaVersion, err := idx.getSchemaVersion(ctx)
	if err != nil {
		return err
	}

	if err := idx.enableForeignKeys(ctx); err != nil {
		return err
	}

	queries := schemaQueries()
	if schemaVersion > len(queries) {
		return fmt.Errorf("database schema version (%d) higher than number of schema queries (%d)", schemaVersion, len(queries))
	}

	for i, stmt := range queries[schemaVersion:] {
		_, err := idx.db.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to apply schema query %d: %w", schemaVersion+i+1, err)
		}
	}

	dlog.Printf("schema CRC: %d", schemaCRC)
	return idx.setUserVersion(ctx, schemaCRC)
}

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
