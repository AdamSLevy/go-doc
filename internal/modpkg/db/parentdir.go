package db

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
)

type ParentDirs struct {
	GOROOT     string
	GOMODCACHE string
	MainModule string
}

func NewParentDirs(GOROOT, GOMODCACHE, mainModuleDir string) ParentDirs {
	return ParentDirs{
		GOROOT:     GOROOT,
		GOMODCACHE: GOMODCACHE,
		MainModule: mainModuleDir,
	}
}

const (
	ParentDirKeyGOROOT     = "GOROOT"
	ParentDirKeyGOMODCACHE = "GOMODCACHE"
	ParentDirKeyMainModule = "main module"
	ParentDirKeyVendor     = "vendor"

	ParentDirIdGOROOT     = 1
	ParentDirIdGOMODCACHE = 2
	ParentDirIdMainModule = 3
	ParentDirIdVendor     = 4
)

func ParentDirID(key string) (int64, error) {
	switch key {
	case ParentDirKeyGOROOT:
		return ParentDirIdGOROOT, nil
	case ParentDirKeyGOMODCACHE:
		return ParentDirIdGOMODCACHE, nil
	case ParentDirKeyMainModule:
		return ParentDirIdMainModule, nil
	case ParentDirKeyVendor:
		return ParentDirIdVendor, nil
	default:
		return -1, fmt.Errorf("unknown parent dir key %q", key)
	}
}

func (pd *ParentDirs) rows() []parentDir {
	return []parentDir{{
		ID:  ParentDirIdGOROOT,
		Key: ParentDirKeyGOROOT,
		Dir: pd.GOROOT,
	}, {
		ID:  ParentDirIdGOMODCACHE,
		Key: ParentDirKeyGOMODCACHE,
		Dir: pd.GOMODCACHE,
	}, {
		ID:  ParentDirIdMainModule,
		Key: ParentDirKeyMainModule,
		Dir: pd.MainModule,
	}, {
		ID:  ParentDirIdVendor,
		Key: ParentDirKeyVendor,
		Dir: filepath.Join(pd.MainModule, "vendor"),
	}}
}

type parentDir struct {
	ID  int64
	Key string
	Dir string
}

const upsertParentDirQuery = `
INSERT INTO
  parent_dir (
    rowid,
    key,
    dir
  )
VALUES (
    ?, ?, ?
)
ON CONFLICT
DO UPDATE SET
  (
    rowid, 
    key, 
    dir
  ) = (
    excluded.rowid,
    excluded.key,
    excluded.dir
  )
;
`

func upsertParentDirs(ctx context.Context, db Querier, dirs *ParentDirs) (rerr error) {
	stmt, err := db.PrepareContext(ctx, upsertParentDirQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare upsert parent dir statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close upsert parent dir statement: %w", err))
		}
	}()

	for _, dir := range dirs.rows() {
		if _, err := stmt.ExecContext(ctx, dir.ID, dir.Key, dir.Dir); err != nil {
			return fmt.Errorf("failed to upsert parent dir %v=%q: %w", dir.Key, dir.Dir, err)
		}
	}
	return nil
}
