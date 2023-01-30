package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type _Tx struct {
	*sql.Tx
}

func (tx *_Tx) Save(ctx context.Context, name string) ReleaseTxFunc {
	if err := savepoint(ctx, tx, name); err != nil {
		panic(fmt.Errorf("sqlite: savepoint: %w", err))
	}
	return func(pErr *error) {
		err := *pErr

	}
}
func savepoint(ctx context.Context, conn QueryContext, name string) error {
	_, err := conn.ExecContext(ctx, "SAVEPOINT "+name)
	return err
}

func (tx *_Tx) Query(query string, args ...any) (Rows, error) {
	return tx.Tx.Query(query, args...)
}
func (tx *_Tx) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return tx.Tx.QueryContext(ctx, query, args...)
}
func (tx *_Tx) QueryRow(query string, args ...any) Row {
	return tx.Tx.QueryRow(query, args...)
}
func (tx *_Tx) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return tx.Tx.QueryRowContext(ctx, query, args...)
}
