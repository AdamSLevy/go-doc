package schema

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

type Class = int

const (
	ClassStdlib Class = iota
	ClassLocal
	ClassRequired
)

func ClassString(c Class) string {
	switch c {
	case ClassStdlib:
		return "stdlib"
	case ClassLocal:
		return "local"
	case ClassRequired:
		return "required"
	default:
		return "unknown class"
	}
}

func ParseClassVendor(root godoc.PackageDir) (Class, bool) {
	if isVendor(root.Dir) {
		return ClassRequired, true
	}
	switch root.ImportPath {
	case "", "cmd":
		return ClassStdlib, false
	}
	if _, hasVersion := parseVersion(root.Dir); hasVersion {
		return ClassRequired, false
	}
	return ClassLocal, false
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}

type Module struct {
	ID         int64
	ImportPath string
	Dir        string
	Class      Class
}

func (s *Sync) initInsertModuleStmt() error {
	const query = `
INSERT INTO 
  main.module (
    import_path,
    dir,
    class
  )
VALUES (
  ?, ?, ?
) 
ON CONFLICT (
  import_path
) 
DO UPDATE SET (
    sync,
    keep
  ) = (
    dir != excluded.dir,
    TRUE
  ), (
    dir, 
    class
  ) = (
    excluded.dir,
    excluded.class
  )
RETURNING 
  rowid,
  sync
;`

	stmt, err := s.tx.PrepareContext(s.ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert module statement: %w", err)
	}

	s.stmt.insertModule = stmt
	return nil
}

func (s *Sync) insertModule(mod *Module) (needSync bool, _ error) {
	row := s.stmt.insertModule.QueryRowContext(s.ctx, mod.ImportPath, mod.Dir, mod.Class)
	if err := row.Err(); err != nil {
		return false, fmt.Errorf("failed to insert module: %w", err)
	}
	if err := row.Scan(&mod.ID, &needSync); err != nil {
		return false, fmt.Errorf("failed to scan inserted module: %w", err)
	}
	return needSync, nil
}

func (s *Sync) pruneModules(ctx context.Context) error {
	_, err := s.tx.ExecContext(ctx, `
DELETE FROM 
  main.module 
WHERE 
  keep=FALSE;
`)
	if err != nil {
		return fmt.Errorf("failed to prune modules: %w", err)
	}
	return nil
}

func SelectAllModules(ctx context.Context, db Querier, mods []Module) (_ []Module, rerr error) {
	return selectModulesFromWhere(ctx, db, mods, "main.module", "")
}
func selectModulesFromWhere(ctx context.Context, db Querier, mods []Module, from, where string, args ...any) (_ []Module, rerr error) {
	stmt, err := selectModulesFromWhereStmt(ctx, db, from, where)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()
	return selectModules(ctx, stmt, mods, args...)
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
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	return stmt, nil
}
func selectModules(ctx context.Context, stmt *sql.Stmt, mods []Module, args ...interface{}) (_ []Module, rerr error) {
	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select modules: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()
	return scanModules(ctx, rows, mods)
}

func scanModules(ctx context.Context, rows *sql.Rows, mods []Module) ([]Module, error) {
	for rows.Next() {
		mod, err := scanModule(rows)
		if err != nil {
			return mods, fmt.Errorf("failed to scan module: %w", err)
		}
		mods = append(mods, mod)
	}
	if err := rows.Err(); err != nil {
		return mods, fmt.Errorf("failed to load next module: %w", err)
	}
	return mods, nil
}

func scanModule(row Scanner) (mod Module, _ error) {
	var (
		importPath sql.NullString
		dir        sql.NullString
		class      sql.NullInt64
	)
	if err := row.Scan(&mod.ID, &importPath, &dir, &class); err != nil {
		return mod, err
	}

	mod.ImportPath = importPath.String
	mod.Dir = dir.String
	mod.Class = Class(class.Int64)

	return mod, nil
}

func (s *Sync) selectModulesPrune() ([]Module, error) {
	return selectModulesFromWhere(s.ctx, s.tx, nil, "main.module", "keep=FALSE ORDER BY rowid")
}
func (s *Sync) selectModulesNeedSync() ([]Module, error) {
	return selectModulesFromWhere(s.ctx, s.tx, nil, "main.module", "sync=TRUE ORDER BY rowid")
}
