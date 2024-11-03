package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"path"

	"aslevy.com/go-doc/internal/sql"
)

type Part struct {
	ID        int64
	Name      string
	ParentID  *int64
	PackageID *int64
	PathDepth int64
}

type PartClosure struct {
	AncestorID   int64
	DescendantID int64
	Depth        int64
}

//go:embed sql/part_select_by_package_id.sql
var querySelectPackageParts string

func SelectPackageParts(ctx context.Context, db sql.Querier, packageID int64, parts []Part) ([]Part, error) {
	rows, err := db.QueryContext(ctx, querySelectPackageParts,
		sql.Named("package_id", packageID),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query parts for packageID: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var part Part
		if err := rows.Scan(&part.ID, &part.Name, &part.ParentID, &part.PackageID); err != nil {
			return nil, fmt.Errorf("failed to scan Part: %w", err)
		}
		parts = append(parts, part)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan all parts: %w", err)
	}
	return parts, nil
}

type ModulePackage struct {
	PackageID         int64
	PackageImportPath string
	Dir               string
}

//go:embed sql/package_select_by_parts.sql
var querySelectPackagesByParts string

func selectPackagesByParts(ctx context.Context, db sql.Querier, parts []string, pkgs []ModulePackage) (_ []ModulePackage, rerr error) {
	rows, err := selectPackagesByPartsRows(ctx, db, true, parts)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close rows: %w", err))
		}
	}()

	for rows.Next() {
		var pkg ModulePackage
		if err := rows.Scan(&pkg.PackageID, &pkg.PackageImportPath, &pkg.Dir); err != nil {
			return nil, fmt.Errorf("failed to scan ModulePackage: %w", err)
		}
		pkgs = append(pkgs, pkg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan all ModulePackages: %w", err)
	}
	return pkgs, nil
}

func (db *DB) SelectPackagesByPartsRows(ctx context.Context, exact bool, parts []string) (*sql.Rows, error) {
	return selectPackagesByPartsRows(ctx, db.db, exact, parts)
}
func selectPackagesByPartsRows(ctx context.Context, db sql.Querier, exact bool, parts []string) (*sql.Rows, error) {
	rows, err := db.QueryContext(ctx, querySelectPackagesByParts,
		sql.Named("search_path", path.Join(parts...)),
		sql.Named("exact", exact),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query packages by parts: %w", err)
	}
	return rows, nil
}
