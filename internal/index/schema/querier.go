package schema

import (
	"context"
	"database/sql"
)

type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	_ Querier = (*sql.DB)(nil)
	_ Querier = (*sql.Conn)(nil)
	_ Querier = (*sql.Tx)(nil)
)

type Scanner interface {
	Scan(dest ...any) error
}

var (
	_ Scanner = (*sql.Row)(nil)
	_ Scanner = (*sql.Rows)(nil)
)
