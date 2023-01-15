package index

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/pager"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/exp/slices"
)

func (pkgIdx *Packages) sync(required ...Module) {
	defer func() { pkgIdx.updatedAt = time.Now() }()

	progressBar := newProgressBar(len(pkgIdx.modules), "syncing modules...")

	unknown := append(moduleList{}, pkgIdx.modules...)
	for _, req := range required {
		var mod *Module
		pos, found := pkgIdx.modules.Search(req)
		if found {
			unknown.Remove(req)
		} else {
			pkgIdx.modules = slices.Insert(pkgIdx.modules, pos, req)
		}
		mod = &pkgIdx.modules[pos]
		if mod.Dir != req.Dir {
			mod.updatedAt = time.Time{} // force rescan
			mod.Dir = req.Dir
		}
		added, removed := mod.sync()
		pkgIdx.syncPartials(*mod, added, removed)
		progressBar.Add(1)
	}

	pkgIdx.modules.Remove(unknown...)
	for _, mod := range unknown {
		pkgIdx.syncPartials(mod, nil, mod.packages)
		progressBar.Add(1)
	}
	progressBar.Finish()
	progressBar.Clear()
}
func (pkgIdx *Packages) syncPartials(mod Module, add, remove packageList) {
	modParts := strings.Split(mod.ImportPath, "/")
	for _, pkg := range remove {
		pkgIdx.partials.Remove(modParts, pkg)
	}
	for _, pkg := range add {
		pkgIdx.partials.Insert(modParts, pkg)
	}
}
func newProgressBar(total int, description string) *progressbar.ProgressBar {
	termMode := outfmt.Format == outfmt.Term && pager.IsTTY(os.Stderr)
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

func (mod *Module) sync() (added, removed packageList) {
	debug.Printf("syncing module %q in %s", mod.ImportPath, mod.Dir)
	defer func() {
		mod.packages.Remove(removed...)
		mod.packages.Insert(added...)
		mod.updatedAt = time.Now()
	}()

	inModule := true
	mod.Dir = filepath.Clean(mod.Dir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []_Package{}
	// next is the queue of directories to examine in the next pass.
	next := []_Package{{
		ImportPathParts: []string{mod.ImportPath},
	}}

	// Assume everything is removed...
	removed = append(removed, mod.packages...)

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
				if inModule {
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
				pos, found := removed.Search(pkg)
				if !found {
					debug.Printf("adding package %q in %s", pkg, dir)
					added.Insert(pkg)
				} else {
					debug.Printf("keeping package %q in %s", pkg, dir)
					removed = slices.Delete(removed, pos, pos+1)
				}
			}
		}
	}
	return
}
