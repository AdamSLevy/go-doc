package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/sql"
)

type DB struct {
	db *sql.DB

	meta *Metadata
	dirs ParentDirs
}

func (db *DB) Close() error {
	return db.db.Close()
}

const goDocDBPath = ".go-doc/modpkg.sqlite3"

func OpenDB(ctx context.Context, GOROOT, GOMODCACHE, GOMOD string) (_ *DB, rerr error) {
	mainModDir := strings.TrimSuffix(GOMOD, "/go.mod")
	dbPath := filepath.Join(mainModDir, goDocDBPath)
	if err := ensureDBPathDir(dbPath); err != nil {
		return nil, err
	}

	parentDirs := NewParentDirs(GOROOT, GOMODCACHE, mainModDir)
	db, err := openDB(ctx, dbPath, parentDirs)
	if err == nil {
		// The database is ready to use.
		return db, nil
	}
	if !errors.Is(err, errSchemaChecksumMismatch) {
		// The error is not a schema checksum mismatch, so we can't
		// recover.
		return nil, err
	}

	// The schema checksum mismatch means that the database schema is
	// incompatible with the current version of the code. We need to remove
	// the database and re-build it. We'll just rename it to be safe.
	if err := os.Rename(dbPath, dbPath+".old"); err != nil {
		return nil, fmt.Errorf("failed to remove existing database with incompatible schema: %w", err)
	}

	return openDB(ctx, dbPath, parentDirs)
}

func openDB(ctx context.Context, dbPath string, parentDirs ParentDirs) (_ *DB, rerr error) {
	sqldb, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", rerr)
	}
	// We need to close the database if we fail to initialize it.
	defer func() {
		if rerr == nil {
			// Success, so leave the database open.
			return
		}
		if err := sqldb.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close database: %w", err))
		}
	}()

	db := DB{
		db:   sqldb,
		dirs: parentDirs,
	}

	if err := db.initialize(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &db, nil
}
func ensureDBPathDir(dbPath string) error {
	dirPath := filepath.Dir(dbPath)
	if err := os.Mkdir(dirPath, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create database directory %s: %w", dirPath, err)
	}
	return nil
}

func (db *DB) initialize(ctx context.Context) (rerr error) {
	ready, err := db.checkSchema(ctx)
	if err != nil {
		return err
	}

	// Always enable foreign keys and recursive triggers.
	if err := db.enableForeignKeys(ctx); err != nil {
		return err
	}
	if err := db.enableRecursiveTriggers(ctx); err != nil {
		return err
	}

	if !ready {
		// The WAL journal mode is persistent so we only need to set it
		// if the database is not ready. This must occur outside of the
		// following transaction.
		if err := db.journalModeWAL(ctx); err != nil {
			return err
		}
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.RollbackOnError(&rerr)

	if !ready {
		if err := applySchema(ctx, tx); err != nil {
			return err
		}
	}

	if err := upsertParentDirs(ctx, tx, &db.dirs); err != nil {
		return err
	}

	if ready {
		if db.meta, err = selectMetadata(ctx, tx); err != nil &&
			!errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	return tx.Commit()
}

var errSchemaChecksumMismatch = errors.New("schema checksum mismatch")

func (db *DB) checkSchema(ctx context.Context) (ready bool, _ error) {
	appID, err := getApplicationID(ctx, db.db)
	if err != nil {
		return false, err
	}

	userVersion, err := getUserVersion(ctx, db.db)
	if err != nil {
		return false, err
	}

	schemaVersion, err := getSchemaVersion(ctx, db.db)
	if err != nil {
		return false, err
	}

	if appID+userVersion+schemaVersion == 0 {
		// Database is uninitialized.
		return false, nil
	}

	if appID != sqliteApplicationID {
		return false, fmt.Errorf("unrecognized database application ID")
	}

	if userVersion != schemaChecksum {
		return false, errSchemaChecksumMismatch
	}

	return true, nil
}
