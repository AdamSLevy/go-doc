package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/dlog"
)

type _Tx struct {
	*sql.Tx
}

func beginTx(ctx context.Context, db *sql.DB) (_Tx, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return _Tx{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return _Tx{tx}, nil
}

func (tx _Tx) RollbackOrCommit(rerr *error) {
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
func (tx _Tx) RollbackOnError(rerr *error) {
	if *rerr == nil {
		return
	}
	dlog.Output(0, "rolling back...")
	if err := tx.Rollback(); err != nil {
		*rerr = errors.Join(*rerr, fmt.Errorf("failed to rollback transaction: %w", err))
	}
}
