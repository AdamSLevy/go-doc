package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Tx is a wrapper around *sql.Tx that provides a RollbackOnError method
// which can be deferred to rollback the transaction depending on the returned
// error.
//
// The Rollback and Commit methods ignore the sql.ErrTxDone error and otherwise
// wrap the returned error if not nil.
type Tx struct{ *sql.Tx }

func (tx *Tx) Exec(query string, args ...any) (Result, error) {
	return tx.ExecContext(context.Background(), query, args...)
}
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return _ExecContext(tx.Tx, ctx, query, args...)
}

func (tx *Tx) Prepare(query string) (*Stmt, error) {
	return tx.PrepareContext(context.Background(), query)
}
func (tx *Tx) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	return _PrepareContext(tx.Tx, ctx, query)
}

func (tx *Tx) Query(query string, args ...any) (*Rows, error) {
	return tx.QueryContext(context.Background(), query, args...)
}
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	return _QueryContext(tx.Tx, ctx, query, args...)
}

func (tx *Tx) QueryRow(query string, args ...any) *Row {
	return tx.QueryRowContext(context.Background(), query, args...)
}
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	row := tx.Tx.QueryRowContext(ctx, query, args...)
	return &Row{row, query, args}
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
func (tx *Tx) RollbackOnError(rerr *error) {
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
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil && !errors.Is(err, ErrTxDone) {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// Commit calls tx.Tx.Commit() and wraps the returned error if not nil.
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (tx *Tx) Stmt(stmt *Stmt) *Stmt { return tx.StmtContext(context.Background(), stmt) }
func (tx *Tx) StmtContext(ctx context.Context, stmt *Stmt) *Stmt {
	return &Stmt{tx.Tx.StmtContext(ctx, stmt.Stmt), stmt.query}
}
