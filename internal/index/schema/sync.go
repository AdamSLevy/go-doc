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
	tx  _Tx
	g   *errgroup.Group
	ctx context.Context

	stmt struct {
		insertModule *sql.Stmt
	}

	pkgs chan Package
}

func NewSync(ctx context.Context, db *sql.DB) (_ *Sync, rerr error) {
	tx, err := beginTx(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("failed to being transaction: %w", err)
	}
	defer tx.RollbackOnError(&rerr)

	g, ctx := errgroup.WithContext(ctx)
	s := Sync{
		tx:  tx,
		g:   g,
		ctx: ctx,
	}

	if err := s.initInsertModuleStmt(); err != nil {
		return nil, err
	}

	if err := s.setAllModuleSyncKeepFalse(); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Sync) setAllModuleSyncKeepFalse() error {
	const query = `
UPDATE
  main.module
SET
  sync = FALSE,
  keep = FALSE;
`
	_, err := s.tx.ExecContext(s.ctx, query)
	if err != nil {
		return fmt.Errorf("failed to apply query: %w\n%s\n", err, query)
	}

	return nil
}

func (s *Sync) AddModule(mod *Module) (needSync bool, rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	return s.insertModule(mod)
}

func (s *Sync) AddPackage(pkg Package) error {
	if s.pkgs == nil &&
		s.ctx.Err() == nil {
		s.pkgs = make(chan Package, 1)
		s.g.Go(s.syncPackages)
	}
	select {
	case <-s.ctx.Done():
	case s.pkgs <- pkg:
	}
	return s.ctx.Err()
}

func (s *Sync) syncPackages() (rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	stmt, err := s.insertPackageStmt()
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close insert package statement: %w", err))
		}
	}()
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
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
}

func (s *Sync) Finish(ctx context.Context, meta Metadata) (rerr error) {
	defer func() {
		if err := s.stmt.insertModule.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close insert module statement: %w", err))
		}
	}()

	if s.pkgs != nil {
		close(s.pkgs)
		if err := s.g.Wait(); err != nil {
			return err
		}
	}

	if err := s.finish(ctx, meta); err != nil {
		return fmt.Errorf("failed to finish sync: %w", err)
	}

	return nil
}
func (s *Sync) finish(ctx context.Context, meta Metadata) (rerr error) {
	defer s.tx.RollbackOrCommit(&rerr)
	if err := s.pruneModules(ctx); err != nil {
		return err
	}
	if err := s.prunePackages(ctx); err != nil {
		return err
	}
	if err := s.upsertMetadata(ctx, meta); err != nil {
		return err
	}
	return nil
}
