package index

import (
	"context"
	"log"
	"os"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index/schema"
	"aslevy.com/go-doc/internal/vendored"
)

func (idx *Index) syncVendoredModules(ctx context.Context, sync *schema.Sync, vendorRoot godoc.PackageDir) error {
	const vendor = true
	needsSync, err := sync.AddRequiredModules(schema.Module{
		ImportPath: vendorRoot.ImportPath,
		Dir:        vendorRoot.Dir,
		Class:      schema.ClassLocal,
		Vendor:     true,
	})
	if err != nil {
		return err
	}

	if len(needsSync) == 0 && idx.vendorUnchanged(vendorRoot) {
		return nil
	}

	return vendored.Parse(ctx, vendorRoot.Dir, func(ctx context.Context, mod godoc.PackageDir, pkgs ...godoc.PackageDir) error {
		needsSync, err := sync.AddRequiredModules(schema.Module{
			ImportPath: mod.ImportPath,
			Dir:        mod.Dir,
			Class:      schema.ClassLocal,
			Vendor:     true,
		})
		if err != nil {
			return err
		}
		if len(needsSync) == 0 {
			return nil
		}
		for _, pkg := range pkgs {
			err := sync.AddPackages(schema.Package{
				ModuleID:     needsSync[0].ID,
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
