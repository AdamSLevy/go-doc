package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index"
)

func packageIndex() *index.Packages {
	localModuleRoot := moduleRootDir(goCmd())
	if localModuleRoot == "" {
		return nil
	}
	path := indexCachePath(localModuleRoot)
	if err := os.Mkdir(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		dlog.Printf("failed to create index cache dir: %v", err)
		return nil
	}
	pkgIdx, err := index.LoadSync(path, dirsToIndexModules(codeRoots()...), index.WithMode(index.Sync))
	if err != nil {
		dlog.Printf("index.UpdateOrCreate: %v", err)
	}
	return pkgIdx
}
func indexCachePath(localModuleRoot string) string {
	return filepath.Join(localModuleRoot, ".go-doc", "index.json")
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
