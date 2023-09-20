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
	ctx context.Context
	tx  *sql.Tx
	g   *errgroup.Group

	stmt struct {
		insertModule *sql.Stmt
	}

	pkgs chan Package
	sync chan Module
}

func NewSync(ctx context.Context, db *sql.DB) (_ *Sync, rerr error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to being transaction: %w", err)
	}
	defer RollbackOnError(tx, &rerr)

	g, ctx := errgroup.WithContext(ctx)
	s := Sync{
		ctx: ctx,
		tx:  tx,
		g:   g,
	}

	if err := s.init(); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Sync) init() error {
	const query = `
UPDATE
  main.module
SET
  sync = FALSE,
  keep = FALSE;---
UPDATE
  main.package
SET
  keep = TRUE;---
`
	if err := execSplit(s.ctx, s.tx, []byte(query)); err != nil {
		return err
	}

	if err := s.initInsertModuleStmt(); err != nil {
		return err
	}

	return nil
}

func (s *Sync) AddRequiredModules(mods ...Module) (needSync []Module, rerr error) {
	defer RollbackOnError(s.tx, &rerr)
	for _, mod := range mods {
		sync, err := s.insertModule(&mod)
		if err != nil {
			return nil, err
		}
		if sync {
			needSync = append(needSync, mod)
		}
	}

	return needSync, nil
}

func (s *Sync) AddPackages(pkgs ...Package) error {
	if s.pkgs == nil {
		s.pkgs = make(chan Package, len(pkgs))
		s.g.Go(s.syncPackages)
	}
	for _, pkg := range pkgs {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case s.pkgs <- pkg:
		}
	}
	return nil
}
func (s *Sync) syncPackages() (rerr error) {
	defer RollbackOnError(s.tx, &rerr)
	stmt, err := s.insertPackageStmt()
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close insert package statement: %w", err))
		}
	}()
	for s.ctx.Err() == nil {
		select {
		case <-s.ctx.Done():
		case pkg, ok := <-s.pkgs:
			if !ok {
				return nil
			}
			if err := s.insertPackage(stmt, pkg); err != nil {
				dlog.Printf("failed to insert package %+v: %v", pkg, err)
				return err
			}
		}
	}
	return s.ctx.Err()
}

func (s *Sync) Finish(ctx context.Context) (rerr error) {
	defer func() {
		if err := s.stmt.insertModule.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close insert module statement: %w", err))
		}
	}()

	if s.pkgs != nil {
		close(s.pkgs)
	}
	if err := s.g.Wait(); err != nil {
		return err
	}

	if err := s.finish(ctx); err != nil {
		return fmt.Errorf("failed to finish sync: %w", err)
	}

	return nil
}
func (s *Sync) finish(ctx context.Context) (rerr error) {
	defer RollbackOrCommit(s.tx, &rerr)
	if err := s.pruneModules(ctx); err != nil {
		return err
	}
	if err := s.prunePackages(ctx); err != nil {
		return err
	}
	if err := s.upsertMetadata(ctx); err != nil {
		return err
	}
	return nil
}
