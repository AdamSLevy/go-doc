package index

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/math"
	islices "aslevy.com/go-doc/internal/slices"
)

func (pkgIdx *Packages) needsSync(codeRoots ...godoc.PackageDir) bool {
	switch pkgIdx.mode {
	case SkipSync:
		return false
	case ForceSync:
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

func (pkgIdx *Packages) sync(coderoots ...godoc.PackageDir) {
	if !pkgIdx.needsSync(coderoots...) {
		return
	}
	defer func() { pkgIdx.updatedAt = time.Now() }()
	// progressBar := newProgressBar(len(coderoots), "syncing modules...")
	// defer func() {
	// 	progressBar.Finish()
	// 	progressBar.Clear()
	// }()

	modules := make(moduleList, 0, math.Max(len(pkgIdx.modules), len(coderoots)))
	defer func() { pkgIdx.modules = modules }()

	vendor := false
	for _, root := range coderoots {
		debug.Println("coderoot:", root)
		mod := toModule(root)
		pos, found := pkgIdx.modules.Search(mod)
		if found {
			debug.Println("found")
			mod = pkgIdx.modules[pos]
		}
		if mod.Vendor {
			debug.Println("vendor")
			if vendor {
				panic("multiple vendor modules")
			}
			modules.Insert(pkgIdx.syncVendored(mod)...)
			// progressBar.Add(1)
			vendor = true
			continue
		}
		if pkgIdx.mode == ForceSync || mod.needsSync(root) {
			mod.Dir = root.Dir
			added, removed := mod.sync()
			pkgIdx.syncPartials(mod, added, removed)
		}
		modules.Insert(mod)
		// progressBar.Add(1)
	}

	_, removed := islices.DiffSorted(pkgIdx.modules, modules, compareModules)
	// progressBar.ChangeMax(progressBar.GetMax() + len(removed))
	for _, mod := range removed {
		pkgIdx.syncPartials(mod, nil, mod.Packages)
		// progressBar.Add(1)
	}
}

func (pkgIdx *Packages) syncPartials(mod module, add, remove packageList) {
	modParts := strings.Split(mod.ImportPath, "/")
	for _, pkg := range remove {
		pkgIdx.partials.Remove(modParts, pkg)
	}
	for _, pkg := range add {
		pkgIdx.partials.Insert(modParts, pkg)
	}
}
func newProgressBar(total int, description string) *progressbar.ProgressBar {
	termMode := true
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription("package index: "+description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),               // show current count e.g. 3/5
		progressbar.OptionSetRenderBlankState(true), // render at 0%
		progressbar.OptionClearOnFinish(),           // clear bar when done
		progressbar.OptionUseANSICodes(termMode),
		progressbar.OptionEnableColorCodes(termMode),
	)
}

func (mod module) needsSync(required godoc.PackageDir) bool {
	// Sync when...
	return mod.Dir != required.Dir || // the dir changes
		mod.Class == classLocal || // the module is local
		len(mod.Packages) == 0 || mod.UpdatedAt.IsZero() // the module has no packages
}
func (mod *module) sync() (added, removed packageList) {
	debug.Printf("syncing module %q in %s", mod.ImportPath, mod.PackageDir)
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
		this, next = next, this[0:0]
		for _, pkg := range this {
			dir := pkg.Dir(*mod)
			debug.Printf("scanning package %q in %s", pkg, dir)
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
				pkg.ImportPathParts = append(pkg.ImportPathParts, name)
				debug.Printf("queuing %s", pkg)
				next = append(next, pkg)
			}
			if hasGoFiles {
				pkgs.Insert(pkg)
			}
		}
	}
	return
}
