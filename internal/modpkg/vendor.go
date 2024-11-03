package modpkg

import (
	"context"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
	"aslevy.com/go-doc/internal/vendored"
)

func (modPkg *ModPkg) syncFromVendorDir(ctx context.Context, sync *db.Sync, vendor godoc.PackageDir) error {
	return vendored.Parse(ctx, vendor.Dir, func(ctx context.Context, modDir godoc.PackageDir) (vendored.PackageHandler, error) {
		mod, err := sync.AddModule(ctx, modDir.ImportPath, modDir.Version)
		if err != nil {
			return nil, err
		}
		if mod == nil {
			return nil, nil
		}

		handlePackage := func(ctx context.Context, pkgImportPath string) error {
			return sync.AddPackage(ctx, mod, pkgImportPath)
		}
		return handlePackage, nil
	})
}
