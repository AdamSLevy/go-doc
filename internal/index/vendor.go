package index

import (
	"context"
	"log"
	"os"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkgdb"
	"aslevy.com/go-doc/internal/vendored"
)

func (idx *Index) syncVendoredModules(ctx context.Context, sync *modpkgdb.Sync, vendorRoot godoc.PackageDir) error {
	const vendor = true
	needsSync, err := sync.AddModule(ctx, &modpkgdb.Module{
		ImportPath:  vendorRoot.ImportPath,
		RelativeDir: vendorRoot.Dir,
	})
	if err != nil {
		return err
	}

	if !needsSync && idx.vendorUnchanged(vendorRoot) {
		return nil
	}

	return vendored.Parse(ctx, vendorRoot.Dir, func(ctx context.Context, mod godoc.PackageDir, pkgs ...godoc.PackageDir) error {
		schemaMod := modpkgdb.Module{
			ImportPath:  mod.ImportPath,
			RelativeDir: mod.Dir,
		}
		needsSync, err := sync.AddModule(ctx, &schemaMod)
		if err != nil {
			return err
		}
		if !needsSync {
			return nil
		}
		for _, pkg := range pkgs {
			err := sync.AddPackage(ctx, &modpkgdb.Package{
				ModuleID:     schemaMod.ID,
				RelativePath: pkg.ImportPath,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (idx *Index) vendorUnchanged(vendor godoc.PackageDir) bool {
	info, err := os.Stat(vendor.Dir)
	if err != nil {
		log.Printf("failed to stat %s: %v", vendor.Dir, err)
		return true
	}
	return idx.UpdatedAt.After(info.ModTime())
}
