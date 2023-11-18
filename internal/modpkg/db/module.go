package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/sql"
)

type Module struct {
	godoc.PackageDir
	ID          int64
	RelativeDir string
	ParentDirID int64
}

func (mod *Module) Package(pkg godoc.PackageDir) *Package {
	return &Package{
		ModuleID:   mod.ID,
		PackageDir: pkg,
	}
}

//go:embed sql/module_upsert.sql
var queryUpsertModule string

func prepareUpsertModule(ctx context.Context, db sql.Querier) (*sql.Stmt, error) {
	return db.PrepareContext(ctx, queryUpsertModule)
}

func (s *Sync) upsertModule(ctx context.Context, mod *Module) (needSync bool, _ error) {
	row := s.stmt.upsertModule.QueryRowContext(
		ctx,
		sql.Named("import_path", mod.ImportPath),
		sql.Named("dir", mod.RelativeDir),
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
	query := buildSelectModulesFromWhereQuery(from, where)
	rows, err := db.QueryContext(ctx, query, args...)
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
func buildSelectModulesFromWhereQuery(from, where string) string {
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
	return query
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
	_, err := db.ExecContext(ctx, queryUpdateModuleParentDir,
		sql.Named("vendor", vendor),
		sql.Named("parent_dir_id_vendor", ParentDirIdVendor),
		sql.Named("parent_dir_id_gomodcache", ParentDirIdGOMODCACHE),
	)
	return err
}
