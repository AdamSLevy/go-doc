package index

import (
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"golang.org/x/exp/slices"
)

// Module represents a Go module and its packages.
type Module struct {
	ImportPath string
	Dir        string // Location of the module on disk.

	packages  packageList
	updatedAt time.Time
}

func NewModule(importPath, dir string) Module {
	mod := Module{
		ImportPath: importPath,
		Dir:        dir,
	}
	return mod
}

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
	pos, found := modList.Search(mod)
	switch {
	case add && !found:
		*modList = slices.Insert(*modList, pos, mod)
	case !add && found:
		*modList = slices.Delete(*modList, pos, pos+1)
	}
}
func (modList moduleList) Search(mod Module) (pos int, found bool) {
	return slices.BinarySearchFunc(modList, mod, compareModules)
}
func compareModules(a, b Module) int {
	if cmp := compareModuleClass(a, b); cmp != 0 {
		return cmp
	}
	return stringsCompare(a.ImportPath, b.ImportPath)
}
func compareModuleClass(a, b Module) int { return a.class() - b.class() }

const (
	modStdlib int = iota
	modLocal
	modVendor
	modRequired
)

func (mod Module) class() int {
	switch mod.ImportPath {
	case "", "cmd":
		return modStdlib
	}
	if _, hasVersion := mod.version(); hasVersion {
		return modRequired
	}
	if mod.isVendor() {
		return modVendor
	}
	return modLocal
}
func (mod Module) version() (string, bool) {
	_, version, found := strings.Cut(filepath.Base(mod.Dir), "@")
	return version, found
}
func (mod Module) isVendor() bool { return filepath.Base(mod.Dir) == "vendor" }

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
}

func newPackage(mod Module, importPath string) _Package {
	return _Package{ImportPathParts: parseImportPathParts(mod, importPath)}
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
	pos, found := pkgList.Search(pkg)
	switch {
	case add && !found:
		*pkgList = slices.Insert(*pkgList, pos, pkg)
	case !add && found:
		*pkgList = slices.Delete(*pkgList, pos, pos+1)
	}
}
func (pkgList packageList) Search(pkg _Package) (pos int, found bool) {
	return slices.BinarySearchFunc(pkgList, pkg, comparePackages)
}

func (pkgList *packageList) RemoveDescendentsOf(pkg _Package) {
	first, found := pkgList.FirstChildOf(pkg)
	if !found {
		// No children
		return
	}
	afterLast := (*pkgList)[first:].AfterDescendentsOf(pkg)
	*pkgList = slices.Delete(*pkgList, first, first+afterLast)
}

func (pkgList packageList) ChildrenOf(parent _Package) packageList {
	first, found := pkgList.FirstChildOf(parent)
	if !found {
		return nil
	}
	afterLast := pkgList[first:].AfterChildrenOf(parent)
	return pkgList[first : first+afterLast]
}
func (pkgList packageList) DescendentsOf(parent _Package) packageList {
	first, found := pkgList.FirstChildOf(parent)
	if !found {
		return nil
	}
	afterLast := pkgList[first:].AfterDescendentsOf(parent)
	return pkgList[first : first+afterLast]
}

func (pkgList packageList) FirstChildOf(parent _Package) (pos int, found bool) {
	child := parent
	child.ImportPathParts = append(child.ImportPathParts, "")
	// We should never have a package with an empty import path part, so
	// found should always be false here. But set and return the found
	// value anyway from this search just in case.
	pos, found = pkgList.Search(child)
	if pos == len(pkgList) {
		// No children
		return
	}
	found = pkgList[pos].IsDescendentOf(parent)
	return
}
func (pkgList packageList) AfterChildrenOf(parent _Package) (pos int) {
	child := parent
	child.ImportPathParts = append(child.ImportPathParts, string(unicode.MaxRune))
	pos, _ = pkgList.Search(child)
	return
}
func (pkgList packageList) AfterDescendentsOf(parent _Package) (pos int) {
	child := parent
	for child.IsDescendentOf(parent) {
		child.ImportPathParts = append(child.ImportPathParts, string(unicode.MaxRune))
		pos += pkgList[pos:].AfterChildrenOf(child)
		if pos >= len(pkgList) {
			return
		}
		child = pkgList[pos]
	}
	return
}

func (child _Package) IsDescendentOf(pkg _Package) bool {
	if len(child.ImportPathParts) < len(pkg.ImportPathParts) {
		return false
	}
	child.ImportPathParts = child.ImportPathParts[:len(pkg.ImportPathParts)]
	return comparePackages(pkg, child) == 0
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
//  1. Module import path, lexicographically.
//  2. Package import path depth, ascending. (e.g. "a/b/c" is less than
//     "a/b/a/a")
//  3. Lexicographic order of the package import path segments, as
//     implemented by slices.CompareFunc.
func comparePackages(a, b _Package) int {
	// 1. Module import path, lexicographically.
	if cmp := stringsCompare(a.ModulePath(), b.ModulePath()); cmp != 0 {
		return cmp
	}
	// 2. Package import path depth, ascending.
	if cmp := len(a.ImportPathParts) - len(b.ImportPathParts); cmp != 0 {
		return cmp
	}
	// 3. Lexicographic order of the package import path segments.
	return slices.CompareFunc(a.ImportPathParts, b.ImportPathParts, stringsCompare)
}
