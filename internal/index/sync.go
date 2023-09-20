package index

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index/schema"
	"golang.org/x/sync/errgroup"
)

var dlogSync = dlog.Child("sync")

func (idx *Index) needsSync(ctx context.Context) (bool, error) {
	switch idx.options.mode {
	case ModeOff, ModeSkipSync:
		return false, nil
	case ModeForceSync:
		return true, nil
	}

	meta, err := schema.NewMetadata(idx.options.mainModPath)
	if err != nil {
		return false, err
	}

	dbMeta, err := schema.SelectMetadata(ctx, idx.db)
	if ignoreErrNoRows(err) != nil {
		return false, err
	}
	meta.CreatedAt, meta.UpdatedAt = dbMeta.CreatedAt, dbMeta.UpdatedAt
	idx.Metadata = meta
	if meta != dbMeta {
		return true, nil
	}

	dlogSync.Printf("created at: %v", dbMeta.CreatedAt.Local())
	dlogSync.Printf("updated at: %v", dbMeta.UpdatedAt.Local())
	return time.Since(dbMeta.UpdatedAt) > idx.options.resyncInterval, nil
}
func ignoreErrNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (idx *Index) syncCodeRoots(ctx context.Context, codeRoots []godoc.PackageDir) (retErr error) {
	needsSync, err := idx.needsSync(ctx)
	if err != nil {
		return err
	}
	if !needsSync {
		return nil
	}

	sync, err := schema.NewSync(ctx, idx.db)
	if err != nil {
		return err
	}
	defer func() {
		retErr = errors.Join(retErr, sync.Finish(ctx, idx.Metadata))
	}()

	pb := newProgressBar(idx.options, len(codeRoots)+1, "syncing code roots")
	defer pb.Finish()

	syncNext := make(chan schema.Module, len(codeRoots))
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer close(syncNext)
		for _, codeRoot := range codeRoots {
			mod := codeRootToModule(codeRoot)
			needSync, err := sync.AddModule(&mod)
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
func codeRootToModule(codeRoot godoc.PackageDir) schema.Module {
	class, _ := schema.ParseClassVendor(codeRoot)
	return schema.Module{
		ImportPath: codeRoot.ImportPath,
		Dir:        codeRoot.Dir,
		Class:      class,
	}
}

func (idx *Index) syncModulePackages(ctx context.Context, sync *schema.Sync, root schema.Module) error {
	dlogSync.Printf("syncing module packages for %q in %q", root.ImportPath, root.Dir)
	root.Dir = filepath.Clean(root.Dir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []schema.Module{}
	// next is the queue of directories to examine in the next pass.
	next := []schema.Module{root}

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
						relativePath := strings.TrimPrefix(pkg.ImportPath[len(root.ImportPath):], "/")
						sPkg := schema.Package{
							ModuleID:     root.ID,
							RelativePath: relativePath,
							ImportPath:   path.Join(pkg.ImportPath, relativePath),
						}
						if err := sync.AddPackage(sPkg); err != nil {
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
				subPkg := schema.Module{
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
