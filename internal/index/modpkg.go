package index

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/godoc"
	islices "aslevy.com/go-doc/internal/slices"
)

// module represents a Go module and its packages.
type module struct {
	godoc.PackageDir

	Class  class // classStdlib, classRequired, or classLocal.
	Vendor bool  // True if Dir is the vendor directory, or a subdirectory.

	Packages  packageList
	UpdatedAt time.Time
}

func newModule(importPath, dir string) module {
	return toModule(godoc.NewPackageDir(importPath, dir))
}
func toModule(pkg godoc.PackageDir) module {
	class, vendor := parseClassVendor(pkg.ImportPath, pkg.Dir)
	mod := module{
		PackageDir: pkg,
		Class:      class,
		Vendor:     vendor,
	}
	return mod
}
func parseClassVendor(importPath, dir string) (class, bool) {
	if isVendor(dir) {
		return classRequired, true
	}
	switch importPath {
	case "", "cmd":
		return classStdlib, false
	}
	if _, hasVersion := parseVersion(dir); hasVersion {
		return classRequired, false
	}
	return classLocal, false
}
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }

func (mod module) shouldOmit() bool { return len(mod.Packages) == 0 && !mod.Vendor }
func (mod module) newPackage(importPath string) _Package {
	return _Package{
		ImportPathParts: parseImportPathParts(mod, importPath),
		Class:           mod.Class,
	}
}
func (mod *module) addPackages(importPaths ...string) {
	for _, importPath := range importPaths {
		mod.Packages.Insert(mod.newPackage(importPath))
	}
}

type moduleList []module

func (modList moduleList) MarshalJSON() ([]byte, error) { return omitEmptyElementsMarshalJSON(modList) }

func (modList *moduleList) Remove(mods ...module) { modList.Update(false, mods...) }
func (modList *moduleList) Insert(mods ...module) { modList.Update(true, mods...) }
func (modList *moduleList) Update(add bool, mods ...module) {
	for _, pkg := range mods {
		modList.update(add, pkg)
	}
}
func (modList *moduleList) update(add bool, mod module) {
	opts := []islices.Option[module]{islices.WithKeepOriginal[module]()}
	if !add {
		opts = append(opts, islices.WithDelete[module]())
	}
	*modList, _, _ = islices.UpdateSorted(*modList, mod, compareModules, opts...)
}
func (modList moduleList) Search(mod module) (pos int, found bool) {
	return slices.BinarySearchFunc(modList, mod, compareModules)
}
func compareModules(a, b module) int {
	if cmp := compareClasses(a.Class, b.Class); cmp != 0 {
		return cmp
	}
	return stringsCompare(a.ImportPath, b.ImportPath)
}
func compareClasses(a, b class) int { return int(a - b) }

const (
	classStdlib class = iota
	classLocal
	classRequired
)

type class int

func (c class) String() string {
	switch c {
	case classStdlib:
		return "stdlib"
	case classLocal:
		return "local"
	case classRequired:
		return "required"
	}
	return "unknown"
}

// _Package is an internal representation of a package used for sorting.
type _Package struct {
	// ImportPathParts contains the full import path of the module in the
	// first element of the slice. The subsequent elements in the slice
	// contain the remaining package import path segments.
	//
	// e.g. For the package "github.com/my/module/a/b/c" in the module
	// "github.com/my/module", the slice would be:
	//
	//   []string{"github.com/my/module", "a", "b", "c"}
	//
	// See comparePackages for the rationale behind this representation.
	ImportPathParts []string
	Class           class
}

func parseImportPathParts(mod module, pkgImportPath string) []string {
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
func (pkg _Package) Dir(mod module) string {
	return filepath.Join(mod.Dir, filepath.Join(pkg.ImportPathParts[1:]...))
}

type packageList []_Package

func (pkgList packageList) PackageDirs(mods moduleList) []godoc.PackageDir {
	pkgs := make([]godoc.PackageDir, 0, len(pkgList))
	for _, pkg := range pkgList {
		var mod module
		mod.ImportPath = pkg.ModulePath()
		mod.Class = pkg.Class
		pos, found := mods.Search(mod)
		if !found {
			continue
		}
		pkgs = append(pkgs, godoc.NewPackageDir(pkg.ImportPath(), pkg.Dir(mods[pos])))
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
