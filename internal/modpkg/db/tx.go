package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// sqlTx is a wrapper around *sql.sqlTx that provides a RollbackOnError method
// which can be deferred to rollback the transaction depending on the returned
// error.
//
// The Rollback and Commit methods ignore the sql.ErrTxDone error and otherwise
// wrap the returned error if not nil.
type sqlTx struct{ *sql.Tx }

// beginTx starts a new transaction returned as sqlTx.
func (db *DB) beginTx(ctx context.Context) (*sqlTx, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &sqlTx{tx}, nil
}

// RollbackOnError calls tx.Rollback() if *rerr is not nil or if recovering
// from a panic. This function should be deferred and passed a pointer to the
// caller's final error.
//
//	func doSomething(tx *sqlTx) (rerr error) {
//	    defer tx.RollbackOnError(&rerr)
//	    if err := tx.Exec(...); err != nil {
//	        return err // will rollback
//	    }
//	    return nil     // will not rollback
//	}
//
// The transaction is left open if not rolled back. The user is responsible for
// calling tx.Commit().
func (tx *sqlTx) RollbackOnError(rerr *error) {
	if p := recover(); p != nil {
		*rerr = errors.Join(*rerr, fmt.Errorf("panic: %v", p))
	}
	if *rerr == nil {
		return
	}
	if err := tx.Rollback(); err != nil {
		*rerr = errors.Join(*rerr, err)
	}
}

// Rollback calls tx.Tx.Rollback() and ignores sql.ErrTxDone errors and
// otherwise wraps the returned error if not nil.
func (tx *sqlTx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// Commit calls tx.Tx.Commit() and wraps the returned error if not nil.
func (tx *sqlTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
