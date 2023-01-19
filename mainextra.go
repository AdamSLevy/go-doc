package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/index"
)

type packageFinder struct {
	search  string
	matches []string
	offset  int
	pkgIdx  *index.Packages
}

var pkgFinder *packageFinder

func (pf *packageFinder) FindNextPackage(pkg string) (string, bool) {
	if pf == nil || pf.pkgIdx == nil {
		return findNextPackage(pkg)
	}
	if pf.search != pkg {
		pf.search = pkg
		pf.matches = pf.pkgIdx.Search(pkg, index.SearchExact())
		pf.offset = 0
	}
	if pf.offset < len(pf.matches) {
		pkg := pf.matches[pf.offset]
		pf.offset++
		return pkg, pf.offset < len(pf.matches)
	}
	return "", false
}

func (pf *packageFinder) Reset() {
	if pf == nil || pf.pkgIdx == nil {
		dirs.Reset()
	}
	pf.search = ""
	pf.matches = nil
	pf.offset = 0
}

func newPackageFinder() *packageFinder {
	return &packageFinder{pkgIdx: packageIndex()}
}

func packageIndex() *index.Packages {
	path := indexCachePath()
	if err := os.Mkdir(filepath.Dir(path), 0755); err != nil && !os.IsExist(err) {
		dlog.Printf("failed to create index cache dir: %v", err)
		return nil
	}
	pkgIdx, _ := index.Load(path, dirsToIndexModules(codeRoots()...)...)
	if err := pkgIdx.Save(path); err != nil {
		dlog.Printf("failed to save index cache: %v", err)
	}
	return pkgIdx
}

func dirsToIndexModules(dirs ...Dir) []index.Module {
	mods := make([]index.Module, len(dirs))
	for i, dir := range dirs {
		mods[i] = index.NewModule(dir.importPath, dir.dir)
	}
	return mods
}

func indexCachePath() string { return filepath.Join(moduleRootDir(goCmd()), ".go-doc", "index.json") }
func moduleRootDir(goCmd string) string {
	args := []string{"env", "GOMOD"}
	stdout, err := exec.Command(goCmd, args...).Output()
	if err != nil {
		dlog.Printf("failed to run `%s %s`: %v", goCmd, strings.Join(args, " "), err)
		return ""
	}
	return filepath.Dir(string(bytes.TrimSpace(stdout)))
}
