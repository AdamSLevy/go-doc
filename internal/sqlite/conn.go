package sqlite

import (
	"context"
	"database/sql"
)

type _Conn struct {
	*sql.Conn
}

func toConn(conn *sql.Conn) Conn {
	return &_Conn{conn}
}

func (c *_Conn) Save(ctx context.Context, name string) ReleaseTxFunc {
	return nil
}

func (c *_Conn) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return c.Conn.QueryContext(ctx, query, args...)
}
func (c *_Conn) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return c.Conn.QueryRowContext(ctx, query, args...)
}
