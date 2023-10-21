package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/dlog"
)

type Tx struct {
	*sql.Tx
}

func (db *DB) beginTx(ctx context.Context) (Tx, error) {
	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return Tx{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return Tx{tx}, nil
}

func (tx Tx) RollbackOrCommit(rerr *error) {
	if *rerr != nil {
		dlog.Output(1, "rolling back...")
		if err := tx.Rollback(); err != nil {
			*rerr = errors.Join(*rerr, fmt.Errorf("failed to rollback transaction: %w", err))
		}
		return
	}
	if err := tx.Commit(); err != nil {
		*rerr = fmt.Errorf("failed to commit transaction: %w", err)
	}
}
func (tx Tx) RollbackOnError(rerr *error) {
	if *rerr == nil {
		return
	}
	dlog.Output(0, "rolling back...")
	if err := tx.Rollback(); err != nil {
		*rerr = errors.Join(*rerr, fmt.Errorf("failed to rollback transaction: %w", err))
	}
}
