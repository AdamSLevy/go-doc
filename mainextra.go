package main

import (
	"context"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index"
)

func packageIndex() *index.Index {
	if GOMOD == "" {
		return nil
	}
	pkgIdx, err := index.Load(context.Background(), GOMOD, dirsToIndexModules(codeRoots()...), index.WithMode(index.Sync))
	if err != nil {
		dlog.Printf("index.Load: %v", err)
	}
	return pkgIdx
}
func dirsToIndexModules(dirs ...Dir) []godoc.PackageDir {
	mods := make([]godoc.PackageDir, len(dirs))
	for i, dir := range dirs {
		mods[i] = godoc.NewPackageDir(dir.importPath, dir.dir)
	}
	return mods
}
