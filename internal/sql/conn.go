package sql

import (
	"context"
	"database/sql"
)

type Conn struct{ *sql.Conn }

func (db *DB) Conn(ctx context.Context) (*Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return &Conn{conn}, nil
}

func (conn *Conn) Begin() (*Tx, error) { return conn.BeginTx(context.Background(), nil) }
func (conn *Conn) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {
	tx, err := conn.Conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (conn *Conn) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return _ExecContext(conn.Conn, ctx, query, args...)
}

func (conn *Conn) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	return _PrepareContext(conn.Conn, ctx, query)
}

func (conn *Conn) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	return _QueryContext(conn.Conn, ctx, query, args...)
}

func (conn *Conn) QueryRow(query string, args ...any) *Row {
	return conn.QueryRowContext(context.Background(), query, args...)
}
func (conn *Conn) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	row := conn.Conn.QueryRowContext(ctx, query, args...)
	return &Row{row, query, args}
}
