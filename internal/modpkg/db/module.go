package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/sql"
)

type Module struct {
	ID          int64
	ImportPath  string
	RelativeDir string
	ParentDirID int64
}

//go:embed sql/module_upsert.sql
var queryUpsertModule string

func prepareUpsertModule(ctx context.Context, db sql.Querier) (*sql.Stmt, error) {
	return db.PrepareContext(ctx, queryUpsertModule)
}

func (s *Sync) upsertModule(ctx context.Context, mod *Module) (needSync bool, _ error) {
	row := s.stmt.upsertModule.QueryRowContext(
		ctx,
		mod.ImportPath,
		mod.RelativeDir,
		mod.ParentDirID,
	)
	return needSync, row.Scan(
		&mod.ID,
		&needSync,
	)
}

func SelectAllModules(ctx context.Context, db sql.Querier) (_ []Module, rerr error) {
	return selectModulesFromWhere(ctx, db, "module", "")
}
func selectModulesFromWhere(ctx context.Context, db sql.Querier, from, where string, args ...any) (_ []Module, rerr error) {
	stmt, err := prepareSelectModulesFromWhere(ctx, db, from, where)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()
	return selectModules(ctx, stmt, args...)
}
func prepareSelectModulesFromWhere(ctx context.Context, db sql.Querier, from, where string) (*sql.Stmt, error) {
	query := `
SELECT
  rowid,
  import_path,
  dir,
  class
FROM
  `[1:] // remove leading newline
	query += from
	if where != "" {
		query += `
WHERE
  `
		query += where
	}
	query += ";"

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w\n%s\n", err, query)
	}
	return stmt, nil
}
func selectModules(ctx context.Context, stmt *sql.Stmt, args ...interface{}) (_ []Module, rerr error) {
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select from module: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()
	return scanModules(ctx, rows)
}
func scanModules(ctx context.Context, rows *sql.Rows) (mods []Module, _ error) {
	for rows.Next() {
		mod, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to load next module: %w", err)
	}
	return mods, nil
}
func scanModule(row sql.RowScanner) (mod Module, _ error) {
	if err := row.Scan(&mod.ID, &mod.ImportPath, &mod.RelativeDir, &mod.ParentDirID); err != nil {
		return mod, fmt.Errorf("failed to scan module: %w", err)
	}
	return mod, nil
}

func (s *Sync) selectModulesToPrune(ctx context.Context) ([]Module, error) {
	return selectModulesFromWhere(ctx, s.tx, "module", "keep=FALSE ORDER BY rowid")
}
func (s *Sync) selectModulesThatNeedSync(ctx context.Context) ([]Module, error) {
	return selectModulesFromWhere(ctx, s.tx, "module", "sync=TRUE ORDER BY rowid")
}

//go:embed sql/module_update_parent_dir.sql
var queryUpdateModuleParentDir string

func updateModuleParentDir(ctx context.Context, db sql.Querier, vendor bool) error {
	_, err := db.ExecContext(ctx, queryUpdateModuleParentDir, vendor, ParentDirIdVendor, ParentDirIdGOMODCACHE)
	return err
}
