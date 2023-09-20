package main

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index"
)

func packageIndex() *index.Index {
	localModuleRoot := moduleRootDir(goCmd())
	if localModuleRoot == "" {
		return nil
	}
	pkgIdx, err := index.Load(context.Background(), localModuleRoot, dirsToIndexModules(codeRoots()...), index.WithMode(index.Sync))
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
func moduleRootDir(goCmd string) string {
	args := []string{"env", "GOMOD"}
	stdout, err := exec.Command(goCmd, args...).Output()
	if err != nil {
		dlog.Printf("failed to run `%s %s`: %v", goCmd, strings.Join(args, " "), err)
		return ""
	}
	return filepath.Dir(string(bytes.TrimSpace(stdout)))
}
