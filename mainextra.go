package main

import (
	"context"
	"log"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg"
)

func openModPkg(ctx context.Context) *modpkg.ModPkg {
	if GOMOD == "" {
		dlog.Printf("GOMOD is empty, not using modpkg")
		return nil
	}
	modPkg, err := modpkg.New(ctx,
		buildCtx.GOROOT, GOMODCACHE, GOMOD,
		dirsToIndexModules(codeRoots()...),
	)
	if err != nil {
		log.Fatalf("modpkg.New: %v", err)
		return nil
	}
	return modPkg
}

func dirsToIndexModules(dirs ...Dir) []godoc.PackageDir {
	mods := make([]godoc.PackageDir, len(dirs))
	for i, dir := range dirs {
		mods[i] = godoc.NewPackageDir(dir.importPath, dir.dir)
	}
	return mods
}
