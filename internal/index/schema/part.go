package schema

import "context"

type Part struct {
	ID        int64
	Name      string
	ParentID  int64
	FullPath  string
	Depth     int64
	PackageID int64
}

func InsertPartsForModulePackage(ctx context.Context, db Querier,
	packageID int64, importPath string) error {
	return nil
}
