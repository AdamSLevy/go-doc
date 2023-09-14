package schema

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/dlog"
	"golang.org/x/sync/errgroup"
)

type Sync struct {
	ctx  context.Context
	tx   *sql.Tx
	g    *errgroup.Group
	pkgs chan Package
}

func NewSync(ctx context.Context, db *sql.DB, required []Module) (_ *Sync, needSync []Module, rerr error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer RollbackOnError(tx, &rerr)

	needSync, err = SyncModules(ctx, tx, required)
	if err != nil {
		return nil, nil, err
	}

	g, ctx := errgroup.WithContext(ctx)
	s := Sync{
		ctx:  ctx,
		tx:   tx,
		g:    g,
		pkgs: make(chan Package, 5),
	}
	g.Go(s.sync)

	return &s, needSync, nil
}

func (s *Sync) SyncPackages(ctx context.Context, pkgs ...Package) error {
	for _, pkg := range pkgs {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.ctx.Done():
			return s.ctx.Err()
		case s.pkgs <- pkg:
		}
	}
	return nil
}
func (s *Sync) sync() (rerr error) {
	defer RollbackOnError(s.tx, &rerr)

	stmt, err := insertPackageStmt(s.ctx, s.tx)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()

	for s.ctx.Err() == nil {
		select {
		case <-s.ctx.Done():
		case pkg, ok := <-s.pkgs:
			if !ok {
				return nil
			}
			if err := insertPackage(s.ctx, stmt, pkg); err != nil {
				dlog.Printf("failed to insert package: %v", err)
				return err
			}
		}
	}
	return s.ctx.Err()
}

func (s *Sync) Finish(ctx context.Context) (rerr error) {
	close(s.pkgs)
	if err := s.g.Wait(); err != nil {
		return fmt.Errorf("failed to sync: %w", err)
	}
	defer RollbackOrCommit(s.tx, &rerr)

	if err := syncFinish(ctx, s.tx); err != nil {
		return fmt.Errorf("failed to finish sync: %w", err)
	}

	return nil
}
func syncFinish(ctx context.Context, db Querier) error {
	if err := pruneModules(ctx, db); err != nil {
		return fmt.Errorf("failed to prune modules: %w", err)
	}
	if err := prunePackages(ctx, db); err != nil {
		return fmt.Errorf("failed to prune packages: %w", err)
	}
	return nil
}
