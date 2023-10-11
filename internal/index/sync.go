package index

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkgdb"
)

var dlogSync = dlog.Child("sync")

func (idx *Index) syncCodeRoots(ctx context.Context, codeRoots []godoc.PackageDir) (retErr error) {
	sync, err := idx.db.StartSyncIfNeeded(ctx)
	if err != nil {
		return fmt.Errorf("failed to start sync: %w", err)
	}
	if sync == nil {
		return nil
	}
	defer func() {
		if err := sync.Finish(ctx); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to finish sync: %w", err))
		}
	}()

	pb := newProgressBar(idx.options, len(codeRoots)+1, "syncing code roots")
	defer func() {
		if err := pb.Finish(); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("failed to finish progress bar: %w", err))
		}
	}()

	syncNext := make(chan modpkgdb.Module, len(codeRoots))
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(syncNext)
		for _, codeRoot := range codeRoots {
			mod := codeRootToModule(codeRoot)
			needSync, err := sync.AddModule(ctx, &mod)
			if err != nil {
				return err
			}
			if !needSync {
				pb.Add(1)
				continue
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case syncNext <- mod:
			}
		}
		return nil
	})
	g.Go(func() error {
		for ctx.Err() == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case mod, ok := <-syncNext:
				if !ok {
					return nil
				}
				if err := idx.syncModulePackages(ctx, sync, mod); err != nil {
					return err
				}
				pb.Add(1)
			}
		}
		return ctx.Err()
	})

	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}
func codeRootToModule(codeRoot godoc.PackageDir) modpkgdb.Module {
	return modpkgdb.Module{
		ImportPath:  codeRoot.ImportPath,
		RelativeDir: codeRoot.Dir,
	}
}

func (idx *Index) syncModulePackages(ctx context.Context, sync *modpkgdb.Sync, root modpkgdb.Module) error {
	dlogSync.Printf("syncing module packages for %q in %q", root.ImportPath, root.RelativeDir)
	root.RelativeDir = filepath.Clean(root.RelativeDir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []modpkgdb.Module{}
	// next is the queue of directories to examine in the next pass.
	next := []modpkgdb.Module{root}

	for len(next) > 0 && ctx.Err() == nil {
		dlogSync.Printf("descending")
		this, next = next, this[0:0]
		for _, pkg := range this {
			dlogSync.Printf("walking %q", pkg)
			fd, err := os.Open(pkg.RelativeDir)
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
						relativePath := strings.TrimPrefix(pkg.ImportPath[len(root.ImportPath):], "/")
						sPkg := modpkgdb.Package{
							ModuleID:     root.ID,
							RelativePath: relativePath,
						}
						if err := sync.AddPackage(ctx, &sPkg); err != nil {
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
				if fi, err := os.Stat(filepath.Join(pkg.RelativeDir, name, "go.mod")); err == nil && !fi.IsDir() {
					continue
				}
				// Remember this (fully qualified) directory for the next pass.
				subPkg := modpkgdb.Module{
					ImportPath:  path.Join(pkg.ImportPath, name),
					RelativeDir: filepath.Join(pkg.RelativeDir, name),
				}
				dlogSync.Printf("queuing %q", subPkg.ImportPath)
				next = append(next, subPkg)
			}
		}
	}

	return ctx.Err()
}
