package db

import (
	"context"
	"fmt"
	"path"
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

func SelectPackageParts(ctx context.Context, db Querier, packageID int64, parts []Part) ([]Part, error) {
	const query = `
SELECT rowid, name, parent_id, package_id FROM part WHERE rowid IN (
        SELECT part_id FROM part_package WHERE package_id = ?
) ORDER BY path_depth ASC;
`
	rows, err := db.QueryContext(ctx, query, packageID)
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

func SelectPackagesByParts(ctx context.Context, db Querier, parts []string, pkgs []ModulePackage) ([]ModulePackage, error) {
	const query = `
WITH RECURSIVE 
  matches (
    remaining_path, 
    part_id, 
    path_depth, 
    part_length
  )
AS 
  (
    VALUES (
      ? || '/', 
      NULL, 
      0, 
      0
    )
    UNION
    SELECT 
      substr(remaining_path, instr(remaining_path, '/')+1),
      part.rowid,
      part.path_depth,
      part_length + length(part.name)
    FROM matches, part
    WHERE
      name LIKE substr(remaining_path, 1, instr(remaining_path, '/')-1) || '%' 
    AND (
        part.parent_id = matches.part_id
      OR 
        matches.part_id IS NULL
    )
    AND
      remaining_path != ''
    ORDER BY 
      3 DESC, 
      4 ASC
  )
SELECT 
  package_id, 
  package_import_path,
  dir
FROM 
  matches, part_package USING (part_id), 
  package_view USING (package_id) 
WHERE 
  remaining_path = ''
ORDER BY 
  (total_num_parts - path_depth) ASC,
  total_num_parts ASC,
  part_length ASC
;
`
	rows, err := db.QueryContext(ctx, query, path.Join(parts...))
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
