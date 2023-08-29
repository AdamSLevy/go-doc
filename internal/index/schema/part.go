package schema

import (
	"context"
	"database/sql"
)

type Part struct {
	ID        int64
	Name      string
	ParentID  int64
	PackageID int64
}

type PartClosure struct {
	AncestorID   int64
	DescendantID int64
	Depth        int64
}

func SelectPackageParts(ctx context.Context, tx *sql.Tx, packageID int64) ([]Part, error) {
	const query = `SELECT id, name, parentId, packageId FROM parts WHERE package_id = $1`
	return nil, nil
}
