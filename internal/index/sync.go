package index

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
