package index

import (
	"path"
	"strings"

	"golang.org/x/exp/slices"
)

type packageList []package_
type package_ []string

func newPackage(mod Module, pkgImportPath string) package_ {
	relPath := strings.TrimPrefix(pkgImportPath, mod.ImportPath)
	relPath = strings.Trim(relPath, "/")
	var relParts []string
	if relPath != "" {
		relParts = strings.Split(relPath, "/")
	}
	parts := make([]string, len(relParts)+1)
	parts[0] = mod.ImportPath
	copy(parts[1:], relParts)
	return package_(parts)
}

func (pkg package_) ModulePath() string { return pkg[0] }
func (pkg package_) ImportPath() string { return path.Join(pkg...) }
func (pkg package_) String() string     { return pkg.ImportPath() }

func (pkgs packageList) ImportPaths() (paths []string) {
	paths = make([]string, len(pkgs))
	for i, pkg := range pkgs {
		paths[i] = pkg.ImportPath()
	}
	return
}

func (pkgs packageList) Remove(pkg package_) packageList { return pkgs.Update(pkg, false) }
func (pkgs packageList) Insert(pkg package_) packageList { return pkgs.Update(pkg, true) }
func (pkgs packageList) Update(pkg package_, add bool) packageList {
	pos, found := slices.BinarySearchFunc(pkgs, pkg, comparePackages)
	switch {
	case add && !found:
		return slices.Insert(pkgs, pos, pkg)
	case !add && found:
		return slices.Delete(pkgs, pos, pos+1)
	}
	return pkgs
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
func comparePackages(a, b package_) int {
	if cmp := stringsCompare(a.ModulePath(), b.ModulePath()); cmp != 0 {
		return cmp
	}
	if cmp := len(a) - len(b); cmp != 0 {
		return cmp
	}
	return slices.CompareFunc(a, b, stringsCompare)
}
