package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

type Module struct {
	ID         int64
	ImportPath string
	Dir        string
	Class      Class
	Vendor     bool
}

func SyncModules(ctx context.Context, db Querier, required []Module) (updated []Module, _ error) {
	if err := createTempModuleTable(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create temporary module table: %w", err)
	}
	if err := insertTempModules(ctx, db, required); err != nil {
		return nil, fmt.Errorf("failed to insert temporary modules: %w", err)
	}
	if err := pruneModules(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to prune modules: %w", err)
	}
	updated, err := upsertModules(ctx, db, make([]Module, 0, len(required)))
	if err != nil {
		return nil, fmt.Errorf("failed to upsert modules: %w", err)
	}
	return updated, nil
}

func createTempModuleTable(ctx context.Context, db Querier) error {
	return createTempTable(ctx, db, "module")
}
func createTempTable(ctx context.Context, db Querier, tableName string) error {
	_, err := db.ExecContext(ctx, fmt.Sprintf(`
DROP TABLE IF EXISTS temp."%[1]s";
CREATE TABLE temp."%[1]s" AS 
  SELECT * FROM main."%[1]s" LIMIT 0;
`, tableName))
	return err
}

func insertTempModules(ctx context.Context, db Querier, mods []Module) (rerr error) {
	stmt, err := db.PrepareContext(ctx, `
INSERT INTO temp.module (import_path, dir, class, vendor) VALUES (?, ?, ?, ?);
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
  WHERE import_path NOT IN (
    SELECT import_path FROM temp.module
);
`)
	return err
}

func upsertModules(ctx context.Context, db Querier, mods []Module) (_ []Module, rerr error) {
	rows, err := db.QueryContext(ctx, `
INSERT INTO main.module (import_path, dir, class, vendor)
  SELECT import_path, dir, class, vendor 
    FROM temp.module
    WHERE true
  ON CONFLICT(import_path) 
    DO UPDATE SET
      dir=excluded.dir,
      class=excluded.class,
      vendor=excluded.vendor
    WHERE dir!=excluded.dir
  RETURNING
    rowid, 
    import_path, 
    dir, 
    class, 
    vendor;
`)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert modules: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()

	return scanModules(ctx, rows, mods)
}

func SelectAllModules(ctx context.Context, db Querier, mods []Module) (_ []Module, rerr error) {
	rows, err := db.QueryContext(ctx, `
SELECT rowid, import_path, dir, class, vendor FROM main.module;
`)
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
	return mod, row.Scan(&mod.ID, &mod.ImportPath, &mod.Dir, &mod.Class, &mod.Vendor)
}
