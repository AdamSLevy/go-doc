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
	ClassNotRequired
)

func ClassString(c Class) string {
	switch c {
	case ClassStdlib:
		return "stdlib"
	case ClassLocal:
		return "local"
	case ClassRequired:
		return "required"
	case ClassNotRequired:
		return "not required"
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
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }

type Module struct {
	ID         int64
	ImportPath string
	Dir        string
	Class      Class
	Vendor     bool
}

func SyncModules(ctx context.Context, db Querier, required []Module) (needSync []Module, _ error) {
	if err := createTempModuleTable(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create temporary module table: %w", err)
	}
	if err := insertModules(ctx, db, required); err != nil {
		return nil, fmt.Errorf("failed to insert temporary modules: %w", err)
	}
	if err := pruneModules(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to prune modules: %w", err)
	}
	needSync, err := selectModulesNeedSync(ctx, db, make([]Module, 0, len(required)))
	if err != nil {
		return nil, fmt.Errorf("failed to upsert modules: %w", err)
	}
	return needSync, nil
}

//go:embed sync_init.sql
var syncInitQuery string

func createTempModuleTable(ctx context.Context, db Querier) error {
	_, err := db.ExecContext(ctx, syncInitQuery)
	return err
}

func insertModules(ctx context.Context, db Querier, mods []Module) (rerr error) {
	stmt, err := db.PrepareContext(ctx, `
INSERT INTO main.module (import_path, dir, class, vendor)
  VALUES (?, ?, ?, ?)
  ON CONFLICT(import_path) 
    DO UPDATE SET
      dir=excluded.dir,
      class=excluded.class,
      vendor=excluded.vendor;
`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()

	for _, mod := range mods {
		_, err := stmt.ExecContext(ctx, mod.ImportPath, mod.Dir, mod.Class, mod.Vendor)
		if err != nil {
			return fmt.Errorf("failed to execute prepared statement: %w", err)
		}
	}
	return nil
}

func pruneModules(ctx context.Context, db Querier) error {
	_, err := db.ExecContext(ctx, `
DELETE FROM main.module 
  WHERE rowid IN (
    SELECT rowid FROM temp.module_prune
);
`)
	return err
}

func selectUpdatedModules(ctx context.Context, db Querier, mods []Module) (_ []Module, rerr error) {
	rows, err := db.QueryContext(ctx, `
SELECT rowid, import_path, dir, class, vendor FROM 
    temp.module_need_sync, main.module USING (rowid);
`)
	if err != nil {
		return nil, fmt.Errorf("failed to select updated modules: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()

	return scanModules(ctx, rows, mods)
}

func SelectAllModules(ctx context.Context, db Querier, mods []Module) (_ []Module, rerr error) {
	return selectModulesFromWhere(ctx, db, mods, "main.module", "")
}
func selectModulesFromWhere(ctx context.Context, db Querier, mods []Module,
	from, where string, args ...interface{}) (_ []Module, rerr error) {
	query := `SELECT rowid, import_path, dir, class, vendor FROM ` + from
	if where != "" {
		query += " WHERE " + where
	}
	query += ";"

	rows, err := db.QueryContext(ctx, query)
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
	if mods == nil {
		mods = make([]Module, 0)
	}
	for rows.Next() && rows.Err() == nil {
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
		vendor     sql.NullBool
	)
	if err := row.Scan(&mod.ID, &importPath, &dir, &class, &vendor); err != nil {
		return mod, err
	}

	mod.ImportPath = importPath.String
	mod.Dir = dir.String
	mod.Class = Class(class.Int64)
	mod.Vendor = vendor.Bool

	return mod, nil
}

func selectModulesPrune(ctx context.Context, db Querier, mods []Module) ([]Module, error) {
	return selectModulesFromWhere(ctx, db, mods, "temp.module_prune LEFT JOIN main.module USING (rowid)", "")
}
func selectModulesNeedSync(ctx context.Context, db Querier, mods []Module) ([]Module, error) {
	return selectModulesFromWhere(ctx, db, mods, "temp.module_need_sync, main.module USING (rowid)", "")
}
