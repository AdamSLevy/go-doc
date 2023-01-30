package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"time"
)

type Query interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (Rows, error)
	QueryRow(query string, args ...any) Row
}
type QueryContext interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
}
type PingContext interface {
	PingContext(ctx context.Context) error
}

type BeginTx interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type ReleaseTxFunc func(*error)

type SavepointTx interface {
	Save(ctx context.Context, name string) ReleaseTxFunc
}

type DB interface {
	Query
	QueryContext
	PingContext
	BeginTx
	SavepointTx
	io.Closer

	Begin() (*sql.Tx, error)
	Ping() error
	Driver() driver.Driver
	Stats() sql.DBStats

	Conn(ctx context.Context) (Conn, error)
	SetConnMaxIdleTime(d time.Duration)
	SetConnMaxLifetime(d time.Duration)
	SetMaxIdleConns(n int)
	SetMaxOpenConns(n int)
}

type Conn interface {
	QueryContext
	PingContext
	BeginTx
	SavepointTx
	io.Closer

	Raw(f func(driverConn any) error) (err error)
}

type Tx interface {
	Query
	QueryContext
	SavepointTx

	Stmt(stmt *sql.Stmt) *sql.Stmt
	StmtContext(ctx context.Context, stmt *sql.Stmt) *sql.Stmt
}

type Row interface {
	Err() error
	Scan(dest ...any) error
}

type Rows interface {
	Row
	Close() error
	ColumnTypes() ([]*sql.ColumnType, error)
	Columns() ([]string, error)
	Next() bool
	NextResultSet() bool
}
