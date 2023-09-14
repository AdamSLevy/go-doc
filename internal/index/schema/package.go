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

	ImportPath string
}

func SyncPackages(ctx context.Context, db Querier, pkgs []Package) error {
	if err := insertPackages(ctx, db, pkgs); err != nil {
		return fmt.Errorf("failed to insert packages: %w", err)
	}
	if err := prunePackages(ctx, db); err != nil {
		return fmt.Errorf("failed to prune packages: %w", err)
	}
	return nil
}

func insertPackageStmt(ctx context.Context, db Querier) (*sql.Stmt, error) {
	return db.PrepareContext(ctx, `
INSERT INTO main.package (module_id, relative_path) VALUES (?, ?)
  ON CONFLICT 
    DO UPDATE SET
      module_id = excluded.module_id,
      relative_path = excluded.relative_path;
`)
}

func insertPackage(ctx context.Context, stmt *sql.Stmt, pkg Package) (rerr error) {
	_, err := stmt.ExecContext(ctx, pkg.ModuleID, pkg.RelativePath)
	if err != nil {
		return fmt.Errorf("failed to execute prepared statement: %w", err)
	}
	return nil
}

func insertPackages(ctx context.Context, db Querier, pkgs []Package) (rerr error) {
	stmt, err := insertPackageStmt(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close statement: %w", err))
		}
	}()

	for _, pkg := range pkgs {
		if err := insertPackage(ctx, stmt, pkg); err != nil {
			return err
		}
	}
	return nil
}

func prunePackages(ctx context.Context, db Querier) error {
	_, err := db.ExecContext(ctx, `
DELETE FROM main.package WHERE rowid IN (
        SELECT rowid FROM temp.package_prune
);
`)
	return err
}

func SelectAllPackages(ctx context.Context, db Querier, pkgs []Package) (_ []Package, rerr error) {
	return selectPackagesFromWhere(ctx, db, pkgs, "main.package", "")
}
func SelectModulePackages(ctx context.Context, db Querier, modId int64) (pkgs []Package, rerr error) {
	return selectPackagesFromWhere(ctx, db, pkgs, "main.package", "module_id = ? ORDER BY rowid", modId)
}
func selectPackagesFromWhere(ctx context.Context, db Querier, pkgs []Package,
	from, where string, args ...interface{}) (_ []Package, rerr error) {
	query := `SELECT rowid, module_id, relative_path, num_parts FROM ` + from
	if where != "" {
		query += " WHERE " + where
	}
	query += ";"

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

func selectPackagesPrune(ctx context.Context, db Querier, pkgs []Package) ([]Package, error) {
	return selectPackagesFromWhere(ctx, db, pkgs, "temp.package_prune, main.package USING (rowid)", "true ORDER BY rowid")
}
