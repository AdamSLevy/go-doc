package db

import (
	"context"
	_ "embed"
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

func SelectPackagesByParts(ctx context.Context, db sql.Querier, parts []string, pkgs []ModulePackage) ([]ModulePackage, error) {
	rows, err := db.QueryContext(ctx, querySelectPackagesByParts,
		sql.Named("seach_path", path.Join(parts...)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query packages by parts: %w", err)
	}
	defer rows.Close()

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
