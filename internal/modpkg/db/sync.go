package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/sql"
)

type Sync struct {
	tx      *sql.Tx
	db      *DB
	Current *Metadata
	stmt    syncStmts
}

type syncStmts struct {
	upsertModule *sql.Stmt
	upsertPkg    *sql.Stmt
}

func (db *DB) Sync(ctx context.Context) (_ *Sync, rerr error) {
	current, err := db.stored.NeedsSync()
	if err != nil {
		return nil, nil
	}
	if current == nil {
		return nil, nil
	}

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to being transaction: %w", err)
	}
	defer tx.RollbackOnError(&rerr)

	upsertModStmt, err := prepareUpsertModule(ctx, tx)
	if err != nil {
		return nil, err
	}

	if err := setAllModuleSyncKeepFalse(ctx, tx); err != nil {
		return nil, err
	}

	return &Sync{
		tx:      tx,
		db:      db,
		Current: current,
		stmt: syncStmts{
			upsertModule: upsertModStmt,
		},
	}, nil
}

//go:embed sql/module_update_set_sync_keep_false.sql
var queryModuleUpdateSetSyncKeepFalse string

func setAllModuleSyncKeepFalse(ctx context.Context, db sql.Querier) error {
	_, err := db.ExecContext(ctx, queryModuleUpdateSetSyncKeepFalse)
	return err
}

func (s *Sync) AddModule(ctx context.Context, importPath, version string) (_ *Module, rerr error) {
	defer s.tx.RollbackOnError(&rerr)

	var mod Module
	mod.ImportPath = importPath
	mod.Version = version

	needSync, err := s.upsertModule(ctx, &mod)
	if err != nil {
		return nil, err
	}
	if !needSync {
		return nil, nil
	}
	if s.stmt.upsertPkg == nil {
		if err := s.prepareStmtUpsertPackage(ctx); err != nil {
			return nil, err
		}
	}

	return &mod, nil
}

func (s *Sync) AddPackage(ctx context.Context, mod *Module, pkgImportPath string) (rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	return s.upsertPackage(ctx, &Package{
		ModuleID:     mod.ID,
		RelativePath: pkgImportPath[len(mod.ImportPath):],
	})
}

func (s *Sync) Finish(ctx context.Context) (rerr error) {
	defer func() {
		if s.stmt.upsertPkg != nil {
			if err := s.stmt.upsertPkg.Close(); err != nil {
				rerr = errors.Join(rerr, fmt.Errorf("failed to close upsert package statement: %w", err))
			}
		}
		if err := s.stmt.upsertModule.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close upsert module statement: %w", err))
		}
	}()

	if err := s.finish(ctx); err != nil {
		return fmt.Errorf("failed to finish sync: %w", err)
	}

	return nil
}
func (s *Sync) finish(ctx context.Context) (rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	if err := s.upsertMetadata(ctx, s.Current); err != nil {
		return err
	}
	return s.tx.Commit()
}
