package sql

import (
	"context"
	"database/sql"
)

type Stmt struct {
	*sql.Stmt
	query string
}

func (stmt *Stmt) Exec(args ...any) (Result, error) {
	return stmt.ExecContext(context.Background(), args...)
}
func (stmt *Stmt) ExecContext(ctx context.Context, args ...any) (Result, error) {
	res, err := stmt.Stmt.ExecContext(ctx, args...)
	if err != nil {
		return nil, newErrFailedQuery(err, "exec", stmt.query, args...)
	}
	return res, nil
}

func (stmt *Stmt) Query(args ...any) (*Rows, error) {
	return stmt.QueryContext(context.Background(), args...)
}
func (stmt *Stmt) QueryContext(ctx context.Context, args ...any) (*Rows, error) {
	rows, err := stmt.Stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, newErrFailedQuery(err, "query", stmt.query, args...)
	}
	return rows, nil
}

func (stmt *Stmt) QueryRow(args ...any) *Row {
	return stmt.QueryRowContext(context.Background(), args...)
}
func (stmt *Stmt) QueryRowContext(ctx context.Context, args ...any) *Row {
	row := stmt.Stmt.QueryRowContext(ctx, args...)
	return &Row{row, stmt.query, args}
}
