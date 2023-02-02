package index

import (
	"context"
	"log"
	"os"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/vendored"
)

func (idx *Index) syncVendoredModules(ctx context.Context, vendorRoot godoc.PackageDir) ([]int64, error) {
	const vendor = true
	modID, needsSync, err := idx.upsertModule(ctx, vendorRoot, classLocal, vendor)
	if err != nil {
		return nil, err
	}

	if !needsSync && idx.vendorUnchanged(vendorRoot) {
		return idx.vendoredModuleIDs(ctx)
	}

	modIDs := []int64{modID}
	if err := vendored.Parse(ctx, vendorRoot.Dir, func(ctx context.Context, mod godoc.PackageDir, pkgs ...godoc.PackageDir) error {
		pkgKeep := make([]int64, len(pkgs))
		modID, _, err := idx.upsertModule(ctx, mod, classRequired, vendor)
		if err != nil {
			return err
		}
		modIDs = append(modIDs, modID)
		for _, pkg := range pkgs {
			pkgID, err := idx.syncPackage(ctx, modID, mod, pkg)
			if err != nil {
				return err
			}
			pkgKeep = append(pkgKeep, pkgID)
		}
		return idx.prunePackages(ctx, modID, pkgKeep)
	}); err != nil {
		return nil, err
	}
	return modIDs, nil
}

func (idx *Index) vendorUnchanged(vendor godoc.PackageDir) bool {
	info, err := os.Stat(vendor.Dir)
	if err != nil {
		log.Printf("failed to stat %s: %v", vendor.Dir, err)
		return true
	}
	return idx.UpdatedAt.After(info.ModTime())
}

func (idx *Index) vendoredModuleIDs(ctx context.Context) ([]int64, error) {
	const query = `SELECT rowid FROM module WHERE vendor = true;`
	rows, err := idx.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modIDs []int64
	for rows.Next() {
		var modID int64
		if err := rows.Scan(&modID); err != nil {
			return nil, err
		}
		modIDs = append(modIDs, modID)
	}

	return modIDs, rows.Err()
}
