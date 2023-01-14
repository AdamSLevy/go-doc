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

	for len(next) > 0 {
		this, next = next, this[0:0]
		for _, pkg := range this {
			dir := pkg.Dir(*mod)
			fd, err := os.Open(dir)
			if err != nil {
				log.Print(err)
				continue
			}
			info, err := fd.Stat()
			if err != nil {
				log.Print(err)
				continue
			}
			if mod.updatedAt.After(info.ModTime()) {
				// This directory and all subdirectories are
				// unchanged since the last time we scanned.
				continue
			}
			entries, err := fd.Readdir(0)
			fd.Close()
			if err != nil {
				log.Print(err)
				continue
			}
			knownChildren := append(packageList{}, mod.packages.ChildrenOf(pkg)...)
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
				next = append(next, pkg)
				knownChildren.Remove(pkg)
			}
			// remaining known children have been removed
			for _, child := range knownChildren {
				removed.Insert(mod.packages.DescendentsOf(child)...)
			}
			_, found := mod.packages.Search(pkg)
			switch {
			case hasGoFiles && !found:
				// New package...
				added.Insert(pkg)
			case !hasGoFiles && found:
				// No longer is a package...
				removed.Insert(pkg)
			}
		}
	}
	return
}
