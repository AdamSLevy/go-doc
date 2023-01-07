package index

import (
	"encoding/json"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/dlog"
)

type Module struct {
	ImportPath string
	Version    string
	Dir        string

	Packages []string `json:",omitempty"`
}

func NewModule(importPath, version, dir string, pkgs ...string) Module {
	return Module{
		ImportPath: importPath,
		Version:    version,
		Dir:        dir,
		Packages:   pkgs,
	}
}

type Packages interface {
	// Sync the index with the given modules by removing any modules not in
	// mods and their packages. Return any modules which differ in version,
	// or are not yet indexed at all.
	//
	// The Modules in mods only need an ImportPath and Version. They do not
	// need their Packages populated.
	//
	// For the index to be fully up to date, all returned outdated Modules
	// must be passed to Update with their Packages populated.
	Sync(mods ...Module) (outdated []Module)

	// Update the index with the given mods.
	//
	// Usually this is called with the outdated Modules returned from Sync
	// after populating their Packages.
	//
	// The Packages of each Module in mods are added to the index, or
	// updated if they are already indexed.
	//
	// Modules already indexed but not in mods are not affected.
	Update(mods ...Module)

	// Search for packages matching the given path, which could be a full
	// import path, or some number of right-most path segments.
	//
	// If SearchExact() is passed, then only packages which exactly match
	// the path segments are returned.
	//
	// Otherwise the segments in path are matched as path segment prefixes.
	Search(path string, opts ...SearchOption) (pkgs []string)
}

func New(mods ...Module) Packages {
	idx := packageIndex{CreatedAt: time.Now()}
	idx.Update(mods...)
	return &idx
}

var _ Packages = (*packageIndex)(nil)

type packageIndex struct {
	Modules    []Module
	ByNumSlash []rightPartialList

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p packageIndex) MarshalJSON() ([]byte, error) {
	p.UpdatedAt = time.Now()
	type _packageIndex packageIndex
	return json.Marshal(_packageIndex(p))
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
	p.updatePackage(mod, pkgImportPath, false)
}
func (p *packageIndex) addPackage(mod Module, pkgImportPath string) {
	p.updatePackage(mod, pkgImportPath, true)
}
func (p *packageIndex) updatePackage(mod Module, pkgImportPath string, add bool) {
	mod.Packages = nil // don't need this
	dlog.Printf("Packages.update(mod:%q, %q, %v)", mod.ImportPath, pkgImportPath, add)
	pkg := newPackage(mod, pkgImportPath)
	var parts []string
	slash := len(pkgImportPath)
	for numSlash := 0; slash >= 0; numSlash++ {
		if len(p.ByNumSlash) == numSlash {
			p.ByNumSlash = append(p.ByNumSlash, rightPartialList{})
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

// rightPartial is a list of packages which all share the same right segments
// of their import paths.
type rightPartial struct {
	// Parts are the right-most segments of the import paths common to all
	// Packages.
	Parts    []string
	Packages packageList
}
type rightPartialList []rightPartial

func (p *rightPartialList) update(parts []string, pkg package_, add bool) {
	dlog.Printf("partials.update(%q, %q)", parts, pkg.ImportPath())
	newPart := rightPartial{
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
func comparePartials(a, b rightPartial) int {
	return slices.CompareFunc(a.Parts, b.Parts, stringsCompare)
}
func stringsCompare(a, b string) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

func (part *rightPartial) update(pkg package_, add bool) {
	part.Packages = part.Packages.Update(pkg, add)
}
