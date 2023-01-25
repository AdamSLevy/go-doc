package index

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/math"
	islices "aslevy.com/go-doc/internal/slices"
)

func (pkgIdx *Packages) needsSync(codeRoots []godoc.PackageDir) bool {
	switch pkgIdx.mode {
	case ModeSkipSync:
		return false
	case ModeForceSync:
		return true
	}

	// Sync if the code roots have changed.
	if !slices.Equal(pkgIdx.codeRoots, codeRoots) {
		return true
	}

	// Otherwise sync if it's been a while since the last sync to pick up
	// changes to the local module.
	return time.Since(pkgIdx.updatedAt) > pkgIdx.resyncInterval
}

func (pkgIdx *Packages) sync(codeRoots []godoc.PackageDir) (changed bool) {
	if !pkgIdx.needsSync(codeRoots) {
		return
	}
	pb := newProgressBar(pkgIdx.options, len(codeRoots), "syncing...")
	defer func() {
		pb.Finish()
	}()

	modules := make(moduleList, 0, math.Max(len(pkgIdx.modules), len(codeRoots)))
	defer func() {
		pkgIdx.codeRoots = codeRoots
		pkgIdx.modules = modules
		changed = changed || pkgIdx.options.mode == ModeForceSync
	}()

	vendor := false
	for _, root := range codeRoots {
		mod := toModule(root)
		pos, found := pkgIdx.modules.Search(mod)
		if found {
			mod = pkgIdx.modules[pos]
		}
		if mod.Vendor {
			if vendor {
				panic("multiple vendor modules")
			}
			vendor = true

			vendored, vendorChanged := pkgIdx.syncVendored(mod, pb)
			modules.Insert(vendored...)
			changed = changed || vendorChanged
			pb.Add(1)
			continue
		}
		if pkgIdx.mode == ModeForceSync || mod.needsSync(root) {
			mod.Dir = root.Dir
			added, removed := mod.sync()
			changed = pkgIdx.syncPartials(mod, added, removed) || changed
		}
		modules.Insert(mod)
		pb.Add(1)
	}

	_, removed := islices.DiffSorted(pkgIdx.modules, modules, compareModules)
	changed = changed || len(removed) > 0
	pb.ChangeMax(pb.GetMax() + len(removed))
	for _, mod := range removed {
		pkgIdx.syncPartials(mod, nil, mod.Packages)
		pb.Add(1)
	}
	return
}

func (pkgIdx *Packages) syncPartials(mod module, add, remove packageList) (changed bool) {
	if len(add) == 0 && len(remove) == 0 {
		return false
	}
	modParts := strings.Split(mod.ImportPath, "/")
	for _, pkg := range remove {
		pkgIdx.partials.Remove(modParts, pkg)
	}
	for _, pkg := range add {
		pkgIdx.partials.Insert(modParts, pkg)
	}
	return true
}

func (mod module) needsSync(required godoc.PackageDir) bool {
	// Sync when...
	return mod.Dir != required.Dir || // the dir changes
		mod.Class == classLocal || // the module is local
		len(mod.Packages) == 0 || mod.UpdatedAt.IsZero() // the module has no packages
}
func (mod *module) sync() (added, removed packageList) {
	dlog.Printf("syncing module %q in %s", mod.ImportPath, mod.PackageDir)
	pkgs := make(packageList, 0, len(mod.Packages))
	defer func() {
		added, removed = islices.DiffSorted(mod.Packages, pkgs, comparePackages)
		if len(added)+len(removed) > 0 {
			mod.Packages = pkgs
		}
		mod.UpdatedAt = time.Now()
	}()

	mod.Dir = filepath.Clean(mod.Dir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []_Package{}
	// next is the queue of directories to examine in the next pass.
	next := []_Package{mod.newPackage(mod.ImportPath)}

	for len(next) > 0 {
		dlog.Printf("descending")
		this, next = next, this[0:0]
		for _, pkg := range this {
			dlog.Printf("walking %q", pkg)
			dir := pkg.Dir(*mod)
			fd, err := os.Open(dir)
			if err != nil {
				log.Print(err)
				continue
			}

			entries, err := fd.Readdir(0)
			fd.Close()
			if err != nil {
				log.Print(err)
				continue
			}
			hasGoFiles := false
			for _, entry := range entries {
				name := entry.Name()
				// For plain files, remember if this directory contains any .go
				// source files, but ignore them otherwise.
				if !entry.IsDir() {
					if !hasGoFiles && strings.HasSuffix(name, ".go") {
						dlog.Printf("%q has go files", pkg)
						pkgs.Insert(pkg)
						hasGoFiles = true
					}
					continue
				}
				// Entry is a directory.

				// The go tool ignores directories starting with ., _, or named "testdata".
				if name[0] == '.' || name[0] == '_' || name == "testdata" {
					continue
				}
				// When in a module, ignore vendor directories and stop at module boundaries.
				if !mod.Vendor {
					if name == "vendor" {
						continue
					}
					if fi, err := os.Stat(filepath.Join(dir, name, "go.mod")); err == nil && !fi.IsDir() {
						continue
					}
				}
				// Remember this (fully qualified) directory for the next pass.
				pkg := pkg
				pkg.ImportPathParts = append([]string{}, pkg.ImportPathParts...)
				pkg.ImportPathParts = append(pkg.ImportPathParts, name)
				dlog.Printf("queuing %q", pkg)
				next = append(next, pkg)
			}
		}
	}
	return
}
