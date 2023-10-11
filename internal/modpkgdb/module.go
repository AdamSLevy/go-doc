package modpkgdb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
)

type Module struct {
	ID          int64
	ImportPath  string
	RelativeDir string
	ParentDirID int64
}

func (s *Sync) prepareStmtUpsertModule(ctx context.Context) (err error) {
	s.stmt.upsertMod, err = prepareStmtUpsertModule(ctx, s.tx)
	return
}
func prepareStmtUpsertModule(ctx context.Context, db Querier) (*sql.Stmt, error) {
	stmt, err := db.PrepareContext(ctx, upsertModuleQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w\n%s", err, upsertModuleQuery)
	}
	return stmt, nil
}

//go:embed module_upsert.sql
var upsertModuleQuery string

func (s *Sync) upsertModule(ctx context.Context, mod *Module) (needSync bool, _ error) {
	row := s.stmt.upsertMod.QueryRowContext(ctx, mod.ImportPath, mod.RelativeDir, mod.ParentDirID)
	if err := row.Err(); err != nil {
		return false, fmt.Errorf("failed to upsert module: %w", err)
	}
	if err := row.Scan(&mod.ID, &needSync); err != nil {
		return false, fmt.Errorf("failed to scan upserted module: %w", err)
	}
	return needSync, nil
}

func SelectAllModules(ctx context.Context, db Querier) (_ []Module, rerr error) {
	return selectModulesFromWhere(ctx, db, "module", "")
}
func selectModulesFromWhere(ctx context.Context, db Querier, from, where string, args ...any) (_ []Module, rerr error) {
	stmt, err := selectModulesFromWhereStmt(ctx, db, from, where)
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
func selectModulesFromWhereStmt(ctx context.Context, db Querier, from, where string) (*sql.Stmt, error) {
	query := `
SELECT
  rowid,
  import_path,
  dir,
  class
FROM
  `
	query += from
	if where != "" {
		query += " WHERE " + where
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

type errNullValue string

func (e errNullValue) Error() string { return fmt.Sprintf("%s is NULL", string(e)) }

func scanModule(row Scanner) (mod Module, _ error) {
	if err := row.Scan(&mod.ID, &mod.ImportPath, &mod.RelativeDir, &mod.ParentDirID); err != nil {
		return mod, fmt.Errorf("failed to scan module: %w", err)
	}
	return mod, nil
}

func (s *Sync) selectModulesPrune(ctx context.Context) ([]Module, error) {
	return selectModulesFromWhere(ctx, s.tx, "module", "keep=FALSE ORDER BY rowid")
}
func (s *Sync) selectModulesNeedSync(ctx context.Context) ([]Module, error) {
	return selectModulesFromWhere(ctx, s.tx, "module", "sync=TRUE ORDER BY rowid")
}

func updateModuleParentDir(ctx context.Context, db Querier, vendor bool) error {
	const query = `
UPDATE
  module
SET
  parent_dir_id = (
    iif(
      $1,
      $2,
      $3
    )
  )
WHERE
  parent_dir_id = (
    iif(
      NOT $1,
      $2,
      $3
    )
  )
;
`
	if _, err := db.ExecContext(ctx, query, vendor, ParentDirIdVendor, ParentDirIdGOMODCACHE); err != nil {
		return fmt.Errorf("failed to update module parent dir: %w", err)
	}
	return nil
}
