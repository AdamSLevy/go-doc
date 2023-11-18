package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/sql"
)

type Package struct {
	ID           int64
	ModuleID     int64
	RelativePath string
	NumParts     int
}

func (s *Sync) prepareStmtUpsertPackage(ctx context.Context) (err error) {
	s.stmt.upsertPkg, err = prepareStmtUpsertPackage(ctx, s.tx)
	return
}

func prepareStmtUpsertPackage(ctx context.Context, db sql.Querier) (*sql.Stmt, error) {
	stmt, err := db.PrepareContext(ctx, upsertPackageQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare upsert package statement: %w", err)
	}
	return stmt, nil
}

//go:embed sql/package_upsert.sql
var upsertPackageQuery string

func (s *Sync) upsertPackage(ctx context.Context, pkg *Package) error {
	row := s.stmt.upsertPkg.QueryRowContext(ctx, pkg.ModuleID, pkg.RelativePath)
	if err := row.Err(); err != nil {
		return fmt.Errorf("failed to upsert package: %w", err)
	}
	if err := row.Scan(&pkg.ID); err != nil {
		return fmt.Errorf("failed to scan upserted package: %w", err)
	}
	return nil
}

func SelectAllPackages(ctx context.Context, db sql.Querier) ([]Package, error) {
	return selectPackagesFromWhere(ctx, db, "package", "")
}
func SelectModulePackages(ctx context.Context, db sql.Querier, modId int64) ([]Package, error) {
	return selectPackagesFromWhere(ctx, db, "package", "module_id = ? ORDER BY rowid", modId)
}
func selectPackagesFromWhere(ctx context.Context, db sql.Querier, from, where string, args ...interface{}) (_ []Package, rerr error) {
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
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	if err := rows.Err(); err != nil {
		return pkgs, fmt.Errorf("failed to load next package: %w", err)
	}
	return pkgs, nil
}

func scanPackage(row sql.RowScanner) (pkg Package, _ error) {
	if err := row.Scan(&pkg.ID, &pkg.ModuleID, &pkg.RelativePath, &pkg.NumParts); err != nil {
		return pkg, fmt.Errorf("failed to scan package: %w", err)
	}
	return pkg, nil
}

func (s *Sync) selectPackagesPrune(ctx context.Context) ([]Package, error) {
	return selectPackagesFromWhere(ctx, s.tx, "package", "keep=FALSE ORDER BY rowid")
}
