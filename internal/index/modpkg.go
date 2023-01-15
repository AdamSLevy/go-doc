package index

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	islices "aslevy.com/go-doc/internal/slices"
)

// Module represents a Go module and its packages.
type Module struct {
	ImportPath string
	Dir        string // Location of the module on disk.
	class      int

	packages  packageList
	updatedAt time.Time
}

func NewModule(importPath, dir string) Module {
	mod := Module{
		ImportPath: importPath,
		Dir:        dir,
		class:      parseClass(importPath, dir),
	}
	return mod
}
func parseClass(importPath, dir string) int {
	switch importPath {
	case "", "cmd":
		return classStdlib
	}
	if _, hasVersion := parseVersion(dir); hasVersion {
		return classRequired
	}
	if isVendor(dir) {
		return classVendor
	}
	return classLocal
}
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }

func (mod Module) shouldOmit() bool { return len(mod.packages) == 0 }

type moduleList []Module

func (modList moduleList) MarshalJSON() ([]byte, error) { return omitEmptyElementsMarshalJSON(modList) }

func (modList *moduleList) Remove(mods ...Module) { modList.Update(false, mods...) }
func (modList *moduleList) Insert(mods ...Module) { modList.Update(true, mods...) }
func (modList *moduleList) Update(add bool, mods ...Module) {
	if modList == nil {
		// A nil packageList is tolerated to allow
		// packageIndex.searchPackages to search forward without
		// actually creating a packageList.
		return
	}
	for _, pkg := range mods {
		modList.update(add, pkg)
	}
}
func (modList *moduleList) update(add bool, mod Module) {
	opts := []islices.Option[Module]{islices.WithKeepOriginal[Module]()}
	if !add {
		opts = append(opts, islices.WithDelete[Module]())
	}
	*modList, _, _ = islices.UpdateSorted(*modList, mod, compareModules, opts...)
}
func (modList moduleList) Search(mod Module) (pos int, found bool) {
	return slices.BinarySearchFunc(modList, mod, compareModules)
}
func compareModules(a, b Module) int {
	if cmp := compareClasses(a.class, b.class); cmp != 0 {
		return cmp
	}
	return stringsCompare(a.ImportPath, b.ImportPath)
}
func compareClasses(a, b int) int { return a - b }

const (
	classStdlib int = iota
	classLocal
	classVendor
	classRequired
)

// _Package is an internal representation of a package used for sorting.
type _Package struct {
	// ImportPathParts contains the full import path of the module in the
	// first element of the slice. The subsequent elements in the slice
	// contain the remaining package import path, split by "/".
	//
	// e.g. For the package "github.com/my/module/a/b/c" in the module
	// "github.com/my/module", the slice would be:
	//
	//   []string{"github.com/my/module", "a", "b", "c"}
	//
	// See comparePackages for the rationale behind this representation.
	ImportPathParts []string
	Class           int
}

func newPackage(mod Module, importPath string) _Package {
	return _Package{
		ImportPathParts: parseImportPathParts(mod, importPath),
		Class:           mod.class,
	}
}
func parseImportPathParts(mod Module, pkgImportPath string) []string {
	relPath := strings.TrimPrefix(pkgImportPath, mod.ImportPath)
	relPath = strings.Trim(relPath, "/")
	var relParts []string
	if relPath != "" {
		relParts = strings.Split(relPath, "/")
	}
	parts := make([]string, len(relParts)+1)
	parts[0] = mod.ImportPath
	copy(parts[1:], relParts)
	return parts
}

func (pkg _Package) ModulePath() string { return pkg.ImportPathParts[0] }
func (pkg _Package) ImportPath() string { return path.Join(pkg.ImportPathParts...) }
func (pkg _Package) String() string     { return pkg.ImportPath() }
func (pkg _Package) Dir(mod Module) string {
	return filepath.Join(mod.Dir, filepath.Join(pkg.ImportPathParts[1:]...))
}

type packageList []_Package

func (pkgList packageList) Dirs(mods moduleList) []string {
	dirs := make([]string, 0, len(pkgList))
	for _, pkg := range pkgList {
		pos, found := mods.Search(Module{ImportPath: pkg.ModulePath()})
		if !found {
			continue
		}
		dirs = append(dirs, pkg.Dir(mods[pos]))
	}
	return dirs
}
func (pkgList packageList) ImportPaths() []string {
	pkgs := make([]string, len(pkgList))
	for i, pkg := range pkgList {
		pkgs[i] = pkg.ImportPath()
	}
	return pkgs
}

func (pkgList *packageList) Remove(pkgs ..._Package) { pkgList.Update(false, pkgs...) }
func (pkgList *packageList) Insert(pkgs ..._Package) { pkgList.Update(true, pkgs...) }
func (pkgList *packageList) Update(add bool, pkgs ..._Package) {
	if pkgList == nil {
		// A nil packageList is tolerated to allow
		// packageIndex.searchPackages to search forward without
		// actually creating a packageList.
		return
	}
	for _, pkg := range pkgs {
		pkgList.update(add, pkg)
	}
}
func (pkgList *packageList) update(add bool, pkg _Package) {
	opts := []islices.Option[_Package]{islices.WithKeepOriginal[_Package]()}
	if !add {
		opts = append(opts, islices.WithDelete[_Package]())
	}
	*pkgList, _, _ = islices.UpdateSorted(*pkgList, pkg, comparePackages, opts...)
}
func (pkgList packageList) Search(pkg _Package) (pos int, found bool) {
	return slices.BinarySearchFunc(pkgList, pkg, comparePackages)
}

// comparePackages compares two packages and returns -1, 0, or 1 if a is less
// than, equal to, or greater than b.
//
// Packages are ordered similarly to the resolution order of official go doc.
// Official go doc performs a breadth-first walk of each required module's
// packages, in lexicographic order of the module import paths.
//
// This results in packages being ordered by:
//
//  1. Module Class (stdlib, local, vendor, required)
//  2. Module import path, lexicographically.
//  3. Package import path depth, ascending. (e.g. "a/b/c" is less than
//     "a/b/a/a")
//  4. Lexicographic order of the package import path segments, as
//     implemented by slices.CompareFunc.
func comparePackages(a, b _Package) int {
	//  1. Module Class (stdlib, local, vendor, required)
	if cmp := compareClasses(a.Class, b.Class); cmp != 0 {
		return cmp
	}
	// 2. Module import path, lexicographically.
	if cmp := stringsCompare(a.ModulePath(), b.ModulePath()); cmp != 0 {
		return cmp
	}
	// 3. Package import path depth, ascending.
	if cmp := len(a.ImportPathParts) - len(b.ImportPathParts); cmp != 0 {
		return cmp
	}
	// 4. Lexicographic order of the package import path segments.
	return slices.CompareFunc(a.ImportPathParts, b.ImportPathParts, stringsCompare)
}
