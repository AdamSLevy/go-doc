package sqlite

import (
	"context"
	"database/sql"
)

func toDB(db *sql.DB) DB {
	return &_DB{db}
}

type _DB struct {
	*sql.DB
}

func (db *_DB) Conn(ctx context.Context) (Conn, error) {
	conn, err := db.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	return toConn(conn), nil
}

func (db *_DB) Save(ctx context.Context, name string) ReleaseTxFunc {
	return nil
}

func (db *_DB) Query(query string, args ...any) (Rows, error) {
	return db.DB.Query(query, args...)
}
func (db *_DB) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}
func (db *_DB) QueryRow(query string, args ...any) Row {
	return db.DB.QueryRow(query, args...)
}
func (db *_DB) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}
