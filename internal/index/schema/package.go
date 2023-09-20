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

func (s *Sync) insertPackageStmt() (*sql.Stmt, error) {
	const query = `
INSERT INTO 
  main.package (
    module_id, 
    relative_path
  )
VALUES (
  ?, ?
)
ON CONFLICT 
  DO UPDATE SET
    keep=TRUE;
`
	stmt, err := s.tx.PrepareContext(s.ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert package statement: %w", err)
	}
	return stmt, nil
}

func (s *Sync) insertPackage(stmt *sql.Stmt, pkg Package) (rerr error) {
	_, err := stmt.ExecContext(s.ctx, pkg.ModuleID, pkg.RelativePath)
	if err != nil {
		return fmt.Errorf("failed to insert package: %w", err)
	}
	return nil
}

func (s *Sync) prunePackages(ctx context.Context) error {
	const query = `
DELETE FROM 
  main.package 
WHERE
  keep=FALSE;
`
	_, err := s.tx.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prune packages: %w", err)
	}
	return nil
}

func SelectAllPackages(ctx context.Context, db Querier, pkgs []Package) ([]Package, error) {
	return selectPackagesFromWhere(ctx, db, "main.package", "")
}
func SelectModulePackages(ctx context.Context, db Querier, modId int64) ([]Package, error) {
	return selectPackagesFromWhere(ctx, db, "main.package", "module_id = ? ORDER BY rowid", modId)
}
func selectPackagesFromWhere(ctx context.Context, db Querier, from, where string, args ...interface{}) (_ []Package, rerr error) {
	query := `
SELECT 
  rowid, 
  module_id, 
  relative_path, 
  num_parts 
FROM
  `
	query += from
	if where != "" {
		query += `
WHERE
  `
		query += where
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
	return scanPackages(ctx, rows)
}

func scanPackages(ctx context.Context, rows *sql.Rows) (pkgs []Package, _ error) {
	for rows.Next() {
		pkg, err := scanPackage(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan package: %w", err)
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

func (s *Sync) selectPackagesPrune() ([]Package, error) {
	return selectPackagesFromWhere(s.ctx, s.tx, "main.package", "keep=FALSE ORDER BY rowid")
}
