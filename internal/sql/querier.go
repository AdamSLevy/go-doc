package sql

import (
	"context"
	"database/sql"
)

type Querier interface {
	Exec(query string, args ...any) (Result, error)
	Prepare(query string) (*Stmt, error)
	Query(query string, args ...any) (*Rows, error)
	QueryRow(query string, args ...any) *Row

	QuerierContext
}

type QuerierContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
	PrepareContext(ctx context.Context, query string) (*Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *Row
}

type sqlQuerierContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func _ExecContext(q sqlQuerierContext, ctx context.Context, query string, args ...any) (Result, error) {
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, newErrFailedQuery(err, "exec", query, args...)
	}
	return res, nil
}

func _PrepareContext(q sqlQuerierContext, ctx context.Context, query string) (*Stmt, error) {
	stmt, err := q.PrepareContext(ctx, query)
	if err != nil {
		return nil, newErrFailedQuery(err, "prepare", query)
	}
	return &Stmt{stmt, query}, nil
}

func _QueryContext(q sqlQuerierContext, ctx context.Context, query string, args ...any) (*Rows, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, newErrFailedQuery(err, "query", query, args...)
	}
	return rows, nil
}
