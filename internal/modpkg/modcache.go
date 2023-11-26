package modpkg

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
	"aslevy.com/go-doc/internal/progressbar"
	"golang.org/x/sync/errgroup"
)

func (modPkg *ModPkg) syncFromGoModCache(ctx context.Context, progressBar *progressbar.ProgressBar, sync *db.Sync, coderoots []godoc.PackageDir) (rerr error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU()*2 + 1)
	g.Go(func() error {
		for _, root := range coderoots {
			mod, err := sync.AddModule(ctx, root)
			if err != nil {
				return err
			}
			if mod == nil {
				progressBar.Add(1)
				continue
			}

			g.Go(func() error {
				if err := modPkg.syncModulePackages(ctx, sync, mod); err != nil {
					return fmt.Errorf("failed to sync module packages: %w", err)
				}
				progressBar.Add(1)
				return nil
			})
		}
		return nil
	})
	return g.Wait()
}

var dlogSync = dlog.Child("sync")

func (modPkg *ModPkg) syncModulePackages(ctx context.Context, sync *db.Sync, mod *db.Module) error {
	mod.Dir = filepath.Clean(mod.Dir) // because filepath.Join will do it anyway
	dlogSync.Printf("syncing packages for module %q in %q", mod.ImportPath, mod.Dir)

	// this is the queue of directories to examine in this pass.
	this := []godoc.PackageDir{}
	// next is the queue of directories to examine in the next pass.
	next := []godoc.PackageDir{mod.PackageDir}

	for len(next) > 0 && ctx.Err() == nil {
		dlogSync.Printf("descending")
		this, next = next, this[0:0]
		for _, pkg := range this {
			dlogSync.Printf("walking %q", pkg)
			fd, err := os.Open(pkg.Dir)
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
						if err := sync.AddPackage(ctx, mod, pkg.ImportPath); err != nil {
							return err
						}
					}
					continue
				}
				// Entry is a directory.

				// The go tool ignores directories starting with ., _, or named "testdata".
				if name[0] == '.' || name[0] == '_' || name == "testdata" {
					continue
				}
				// Ignore vendor directories and stop at module boundaries.
				if name == "vendor" {
					continue
				}
				if fi, err := os.Stat(filepath.Join(pkg.Dir, name, "go.mod")); err == nil && !fi.IsDir() {
					continue
				}
				// Remember this (fully qualified) directory for the next pass.
				subPkg := godoc.PackageDir{
					ImportPath: path.Join(pkg.ImportPath, name),
					Dir:        filepath.Join(pkg.Dir, name),
				}
				dlogSync.Printf("queuing %q", subPkg.ImportPath)
				next = append(next, subPkg)
			}
		}
	}

	return ctx.Err()
}
