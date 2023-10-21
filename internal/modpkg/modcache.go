package modpkg

import (
	"context"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
)

func (modPkg *ModPkg) syncGoModCache(ctx context.Context, sync *db.Sync, coderoots []godoc.PackageDir) error {
	return nil
}
