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
	ID int64
	godoc.PackageDir
}

//go:embed query/module_upsert.sql
var queryModuleUpsertSql string

func prepareUpsertModule(ctx context.Context, db sql.Querier) (*sql.Stmt, error) {
	return db.PrepareContext(ctx, queryModuleUpsertSql)
}

func (s *Sync) upsertModule(ctx context.Context, mod *Module) (needSync bool, _ error) {
	row := s.stmt.upsertModule.QueryRowContext(
		ctx,
		sql.Named("import_path", mod.ImportPath),
		sql.Named("version", mod.Version),
	)
	return needSync, row.Scan(
		&needSync,
		&mod.ID,
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
	if err := row.Scan(&mod.ID, &mod.ImportPath); err != nil {
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
