package db

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
)

type Sync struct {
	tx   Tx
	Meta Metadata
	stmt struct {
		upsertMod *sql.Stmt
		upsertPkg *sql.Stmt
	}
}

func (db *DB) StartSyncIfNeeded(ctx context.Context) (_ *Sync, rerr error) {
	meta, err := NewMetadata(db.dirs.MainModule)
	if err != nil {
		return nil, err
	}

	if db.meta.BuildRevision == meta.BuildRevision &&
		db.meta.GoVersion == meta.GoVersion &&
		db.meta.GoModHash == meta.GoModHash &&
		db.meta.GoSumHash == meta.GoSumHash {
		if db.meta.Vendor == meta.Vendor {
			return nil, nil
		}

		// If the only thing that changed is use of a vendor directory,
		// then we can just update the parent directory reference for
		// all modules between the GOMODCACHE dir and the vendor dir.
		if err := updateModuleParentDir(ctx, db.db, meta.Vendor); err != nil {
			return nil, err
		}
		return nil, nil
	}

	tx, err := db.beginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to being transaction: %w", err)
	}
	defer tx.RollbackOnError(&rerr)

	s := Sync{
		tx:   tx,
		Meta: meta,
	}

	if err := s.setAllModuleSyncKeepFalse(ctx); err != nil {
		return nil, err
	}

	if err := s.prepareStmtUpsertModule(ctx); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Sync) setAllModuleSyncKeepFalse(ctx context.Context) error {
	const query = `
UPDATE
  module
SET
  sync = FALSE,
  keep = FALSE;
`

	if _, err := s.tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to apply query: %w\n%s\n", err, query)
	}

	return nil
}

func (s *Sync) AddModule(ctx context.Context, mod *Module) (needSync bool, rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	needSync, rerr = s.upsertModule(ctx, mod)
	// if the module requires syncing
	if rerr == nil && needSync &&
		// and we haven't yet prepared the upsert package statement.
		s.stmt.upsertPkg == nil {
		// then prepare the upsert package statement.
		rerr = s.prepareStmtUpsertPackage(ctx)
	}
	return
}

func (s *Sync) AddPackage(ctx context.Context, pkg *Package) (rerr error) {
	defer s.tx.RollbackOnError(&rerr)
	return s.upsertPackage(ctx, pkg)
}

func (s *Sync) Finish(ctx context.Context) (rerr error) {
	defer func() {
		if s.stmt.upsertPkg != nil {
			if err := s.stmt.upsertPkg.Close(); err != nil {
				rerr = errors.Join(rerr, fmt.Errorf("failed to close upsert package statement: %w", err))
			}
		}
		if err := s.stmt.upsertMod.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close upsert module statement: %w", err))
		}
	}()

	if err := s.finish(ctx); err != nil {
		return fmt.Errorf("failed to finish sync: %w", err)
	}

	return nil
}
func (s *Sync) finish(ctx context.Context) (rerr error) {
	defer s.tx.RollbackOrCommit(&rerr)
	if err := s.upsertMetadata(ctx, &s.Meta); err != nil {
		return err
	}
	return nil
}
