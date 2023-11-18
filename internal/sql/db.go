package sql

import (
	"context"
	"database/sql"
)

type DB struct{ *sql.DB }

func Open(driverName, dataSourceName string) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) Begin() (*Tx, error) { return db.BeginTx(context.Background(), nil) }
func (db *DB) BeginTx(ctx context.Context, opts *TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{tx}, nil
}

func (db *DB) Exec(query string, args ...any) (Result, error) {
	return db.ExecContext(context.Background(), query, args...)
}
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (Result, error) {
	return _ExecContext(db.DB, ctx, query, args...)
}

func (db *DB) Prepare(query string) (*Stmt, error) {
	return db.PrepareContext(context.Background(), query)
}
func (db *DB) PrepareContext(ctx context.Context, query string) (*Stmt, error) {
	return _PrepareContext(db.DB, ctx, query)
}

func (db *DB) Query(query string, args ...any) (*Rows, error) {
	return db.QueryContext(context.Background(), query, args...)
}
func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*Rows, error) {
	return _QueryContext(db.DB, ctx, query, args...)
}

func (db *DB) QueryRow(query string, args ...any) *Row {
	return db.QueryRowContext(context.Background(), query, args...)
}
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *Row {
	row := db.DB.QueryRowContext(ctx, query, args...)
	return &Row{row, query, args}
}
