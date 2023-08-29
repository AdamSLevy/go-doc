package schema

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Package struct {
	ID           int64
	ModuleID     int64
	RelativePath string
	NumParts     int
}

func SyncPackages(ctx context.Context, db Querier, pkgs []Package) (updated []Package, _ error) {
	if err := createTempPackageTable(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create temporary package table: %w", err)
	}
	if err := insertTempPackages(ctx, db, pkgs); err != nil {
		return nil, fmt.Errorf("failed to insert temporary packages: %w", err)
	}
	if err := prunePackages(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to prune packages: %w", err)
	}
	updated, err := upsertPackages(ctx, db, make([]Package, 0, len(pkgs)))
	if err != nil {
		return nil, fmt.Errorf("failed to upsert packages: %w", err)
	}
	return updated, nil
}

func createTempPackageTable(ctx context.Context, db Querier) error {
	return createTempTable(ctx, db, "package")
}

func insertTempPackages(ctx context.Context, db Querier, pkgs []Package) (rerr error) {
	stmt, err := db.PrepareContext(ctx, `
INSERT INTO temp.package (module_id, relative_path) VALUES (?, ?);
`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()

	for _, pkg := range pkgs {
		_, err := stmt.ExecContext(ctx, pkg.ModuleID, pkg.RelativePath)
		if err != nil {
			return fmt.Errorf("failed to execute prepared statement: %w", err)
		}
	}
	return nil
}

func prunePackages(ctx context.Context, db Querier) error {
	_, err := db.ExecContext(ctx, `
DELETE FROM main.package 
  WHERE (module_id) IN (
    SELECT DISTINCT module_id FROM temp.package
  ) AND (module_id, relative_path) NOT IN (
    SELECT module_id, relative_path FROM temp.package
  );
`)
	return err
}

func upsertPackages(ctx context.Context, db Querier, pkgs []Package) (_ []Package, rerr error) {
	rows, err := db.QueryContext(ctx, `
INSERT INTO main.package (module_id, relative_path)
  SELECT module_id, relative_path 
    FROM temp.package
    WHERE true
  ON CONFLICT(module_id, relative_path) 
    DO NOTHING
  RETURNING
    rowid, 
    module_id,
    relative_path,
    num_parts;
`)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert packages: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()

	return scanPackages(ctx, rows, pkgs)
}

func SelectAllPackages(ctx context.Context, db Querier, pkgs []Package) (_ []Package, rerr error) {
	return selectPackagesWhere(ctx, db, pkgs, "")
}
func SelectModulePackages(ctx context.Context, db Querier, modId int64) (pkgs []Package, rerr error) {
	return selectPackagesWhere(ctx, db, pkgs, "module_id = ?", modId)
}
func selectPackagesWhere(ctx context.Context, db Querier, pkgs []Package, where string, args ...interface{}) (_ []Package, rerr error) {
	query := `
SELECT rowid, module_id, relative_path, num_parts FROM main.package`
	if where != "" {
		query += " WHERE " + where + ";"
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select packages: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()
	return scanPackages(ctx, rows, pkgs)
}

func scanPackages(ctx context.Context, rows *sql.Rows, pkgs []Package) ([]Package, error) {
	for rows.Next() && rows.Err() == nil {
		pkg, err := scanPackage(rows)
		if err != nil {
			return pkgs, fmt.Errorf("failed to scan package: %w", err)
		}
		pkgs = append(pkgs, pkg)
	}
	if err := rows.Err(); err != nil {
		return pkgs, fmt.Errorf("failed to load next package: %w", err)
	}
	return pkgs, nil
}

func scanPackage(row Scanner) (pkg Package, _ error) {
	return pkg, row.Scan(&pkg.ID, &pkg.ModuleID, &pkg.RelativePath, &pkg.NumParts)
}
