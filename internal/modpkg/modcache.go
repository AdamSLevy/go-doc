package modpkg

import (
	"context"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
)

func (modPkg *ModPkg) syncGoModCache(ctx context.Context, sync *db.Sync, coderoots []godoc.PackageDir) error {
	for _, codeRoot := range coderoots {
		mod := codeRootToModule(codeRoot)
		syncPkgs, err := sync.AddModule(ctx, &mod)
		if err != nil {
			return err
		}
		if !syncPkgs {
			// pb.Add(1)
			continue
		}

		modPkg.syncPackages(ctx, sync, &mod)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case syncNext <- mod:
		}
	}
	return nil
}

func codeRootToModule(codeRoot godoc.PackageDir) db.Module {
	return db.Module{
		Path:    codeRoot.Path,
		Version: codeRoot.Version,
	}
}
