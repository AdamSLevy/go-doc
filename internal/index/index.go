package index

import (
	"path"
	"strings"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/dlog"
)

type Packages interface {
	Sync(mods ...Module) (outdated []Module)
	Update(mods ...Module)
	Search(path string, opts ...SearchOption) (pkgs []string)
}

var _ Packages = (*packageIndex)(nil)

type packageIndex struct {
	Modules    []Module
	ByNumSlash []partialList
}

type partialList []partial
type partial struct {
	Parts    []string
	Packages packageList
}

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

type packageList []package_

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
	case !found && add:
		return slices.Insert(pkgs, pos, pkg)
	case found && !add:
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

type Module struct {
	ImportPath string
	Version    string
	Packages   []string `json:",omitempty"`
}

// Sync the index with the given mods.
//
// Any modules not in mods are completely removed from the index and any
// outdated modules are returned.
//
// A module is considered outdated if it is not yet indexed, or if its version
// is different from what the index currently has.
func (p *packageIndex) Sync(mods ...Module) (outdated []Module) {
	toKeep := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		toKeep[mod.ImportPath] = struct{}{}
		if p.needsUpdate(mod) {
			outdated = append(outdated, mod)
		}
	}
	for _, mod := range p.Modules {
		if _, keep := toKeep[mod.ImportPath]; !keep {
			p.remove(mod)
		}
	}
	return
}
func (p *packageIndex) needsUpdate(mod Module) bool {
	pos, found := slices.BinarySearchFunc(p.Modules, mod, compareModules)
	return !found || p.Modules[pos].Version != mod.Version
}
func compareModules(a, b Module) int { return stringsCompare(a.ImportPath, b.ImportPath) }

func (p *packageIndex) Update(mods ...Module) {
	for _, mod := range mods {
		p.add(mod)
	}
}
func (p *packageIndex) add(mod Module)    { p.updateModule(mod, true) }
func (p *packageIndex) remove(mod Module) { p.updateModule(mod, false) }
func (p *packageIndex) updateModule(mod Module, add bool) {
	var existing Module
	pos, found := slices.BinarySearchFunc(p.Modules, mod, compareModules)
	if found {
		existing = p.Modules[pos]
		if !add {
			mod.Packages = nil // removes all packages
			p.Modules = slices.Delete(p.Modules, pos, pos+1)
		}
	} else {
		p.Modules = slices.Insert(p.Modules, pos, mod)
	}

	// Packages from the pre-existing module
	oldPkgs := make(map[string]struct{}, len(existing.Packages))
	for _, pkg := range existing.Packages {
		oldPkgs[pkg] = struct{}{}
	}

	// Add new packages not already defined by the old module.
	// Prune oldPkgs down to the packages that are not in the new module.
	for _, pkg := range mod.Packages {
		if _, ok := oldPkgs[pkg]; ok {
			delete(oldPkgs, pkg)
			continue
		}
		p.addPackage(mod, pkg)
	}

	// Remove packages that are no longer defined by the module.
	for pkg := range oldPkgs {
		p.removePackage(mod, pkg)
	}
}

func (p *packageIndex) removePackage(mod Module, pkgImportPath string) {
	p.update(mod, pkgImportPath, false)
}
func (p *packageIndex) addPackage(mod Module, pkgImportPath string) {
	p.update(mod, pkgImportPath, true)
}
func (p *packageIndex) update(mod Module, pkgImportPath string, add bool) {
	mod.Packages = nil // don't need this
	dlog.Printf("Packages.update(mod:%q, %q, %v)", mod.ImportPath, pkgImportPath, add)
	pkg := newPackage(mod, pkgImportPath)
	var parts []string
	slash := len(pkgImportPath)
	for numSlash := 0; slash >= 0; numSlash++ {
		if len(p.ByNumSlash) == numSlash {
			p.ByNumSlash = append(p.ByNumSlash, partialList{})
		}
		prevSlash := slash
		slash = strings.LastIndex(pkgImportPath[:slash], "/")
		parts = slices.Insert(parts, 0, pkgImportPath[slash+1:prevSlash])

		p.ByNumSlash[numSlash].update(parts, pkg, add)
	}

	for i := len(p.ByNumSlash) - 1; i >= 0; i-- {
		if len(p.ByNumSlash[i]) > 0 {
			p.ByNumSlash = p.ByNumSlash[:i+1]
			return
		}
	}
}

func (p *partialList) update(parts []string, pkg package_, add bool) {
	dlog.Printf("partials.update(%q, %q)", parts, pkg.ImportPath())
	newPart := partial{
		Parts:    parts,
		Packages: packageList{pkg},
	}
	pos, found := slices.BinarySearchFunc(*p, newPart, comparePartials)
	if found {
		partial := &(*p)[pos]
		partial.update(pkg, add)
		if len(partial.Packages) == 0 {
			*p = slices.Delete(*p, pos, pos+1)
		}
		return
	}
	if add {
		*p = slices.Insert(*p, pos, newPart)
	}
}
func comparePartials(a, b partial) int { return slices.CompareFunc(a.Parts, b.Parts, stringsCompare) }
func stringsCompare(a, b string) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}
func (part *partial) update(pkg package_, add bool) { part.Packages = part.Packages.Update(pkg, add) }
