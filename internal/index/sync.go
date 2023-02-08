package index

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"aslevy.com/go-doc/internal/godoc"
)

var dlogSync = dlog.Child("sync")

func (idx *Index) needsSync(ctx context.Context) (bool, error) {
	switch idx.options.mode {
	case ModeOff, ModeSkipSync:
		return false, nil
	case ModeForceSync:
		return true, nil
	}
	if err := idx.selectSync(ctx); ignoreErrNoRows(err) != nil {
		return false, err
	}
	dlogSync.Printf("created at: %v", idx.sync.CreatedAt.Local())
	dlogSync.Printf("updated at: %v", idx.sync.UpdatedAt.Local())
	return time.Since(idx.sync.UpdatedAt) > idx.options.resyncInterval, nil
}
func ignoreErrNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	return err
}

func (idx *Index) syncCodeRoots(ctx context.Context, codeRoots []godoc.PackageDir) (retErr error) {
	needsSync, err := idx.needsSync(ctx)
	if err != nil || !needsSync {
		return err
	}

	dlogSync.Println("syncing code roots...")
	commitIfNilErr, err := idx.beginTx(ctx)
	if err != nil {
		return err
	}
	defer commitIfNilErr(&retErr)

	pb := newProgressBar(idx.options, len(codeRoots)+1, "syncing code roots")
	defer pb.Finish()

	var keep []int64
	for _, codeRoot := range codeRoots {
		modIDs, err := idx.syncCodeRoot(ctx, codeRoot)
		if err != nil {
			return err
		}
		keep = append(keep, modIDs...)
		pb.Add(1)
	}

	const vendor = false
	if err := idx.pruneModules(ctx, vendor, keep); err != nil {
		return err
	}
	pb.Add(1)

	return idx.upsertSync(ctx)
}
func (idx *Index) beginTx(ctx context.Context) (commitIfNilErr func(*error), _ error) {
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	idx.tx = newSqlTx(tx)
	return func(retErr *error) {
		if *retErr != nil {
			tx.Rollback()
			return
		}
		*retErr = tx.Commit()
	}, nil
}

func (idx *Index) syncCodeRoot(ctx context.Context, root godoc.PackageDir) (modIDs []int64, _ error) {
	class, vendor := parseClassVendor(root)
	if vendor {
		// ImportPath is empty for vendor directories, so we use the
		// Dir instead so as not to conflict with the stdlib, which
		// uses the empty import path.
		//
		// The root vendor module is a place holder, so its import path
		// will never be used to form a package path.
		root.ImportPath = root.Dir
		return idx.syncVendoredModules(ctx, root)
	}
	return idx.syncModule(ctx, root, class)
}

func parseClassVendor(root godoc.PackageDir) (class, bool) {
	if isVendor(root.Dir) {
		return classRequired, true
	}
	switch root.ImportPath {
	case "", "cmd":
		return classStdlib, false
	}
	if _, hasVersion := parseVersion(root.Dir); hasVersion {
		return classRequired, false
	}
	return classLocal, false
}
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }

func (idx *Index) syncModule(ctx context.Context, root godoc.PackageDir, class int) (modIDs []int64, _ error) {
	const vendor = false
	modID, needsSync, err := idx.upsertModule(ctx, root, class, vendor)
	if err != nil {
		return nil, err
	}
	modIDs = append(modIDs, modID)

	if !needsSync {
		dlogSync.Printf("code root %q is already synced", root.ImportPath)
		return modIDs, nil
	}

	return modIDs, idx.syncModulePackages(ctx, modID, root)
}

func (idx *Index) upsertModule(ctx context.Context, root godoc.PackageDir, class class, vendor bool) (modID int64, needsSync bool, _ error) {
	mod, err := idx.selectModule(ctx, root.ImportPath)
	if ignoreErrNoRows(err) != nil {
		return -1, false, err
	}
	if mod.Dir == root.Dir {
		// The module is already in the database and the directory
		// hasn't changed, so we assume we are synced.
		return mod.ID, false, nil
	}

	modID = mod.ID
	if modID < 1 {
		modID, err = idx.insertModule(ctx, root, class, vendor)
		if err != nil {
			return -1, false, err
		}
	} else {
		if err := idx.updateModule(ctx, modID, root, class, vendor); err != nil {
			return -1, false, err
		}
	}
	return modID, true, nil
}

func (idx *Index) syncModulePackages(ctx context.Context, modID int64, root godoc.PackageDir) error {
	dlogSync.Printf("syncing module packages for %q in %q", root.ImportPath, root.Dir)
	root.Dir = filepath.Clean(root.Dir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []godoc.PackageDir{}
	// next is the queue of directories to examine in the next pass.
	next := []godoc.PackageDir{root}

	var keep []int64
	for len(next) > 0 {
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
						pkgID, err := idx.syncPackage(ctx, modID, root, pkg)
						if err != nil {
							return err
						}
						keep = append(keep, pkgID)
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
				subPkg := godoc.NewPackageDir(
					path.Join(pkg.ImportPath, name),
					filepath.Join(pkg.Dir, name),
				)
				dlogSync.Printf("queuing %q", subPkg.ImportPath)
				next = append(next, subPkg)
			}
		}
	}

	return idx.prunePackages(ctx, modID, keep)
}

func (idx *Index) syncPackage(ctx context.Context, modID int64, root, pkg godoc.PackageDir) (int64, error) {
	dlogSync.Printf("syncing package %q in %q", pkg.ImportPath, pkg.Dir)
	relativePath := strings.TrimPrefix(pkg.ImportPath[len(root.ImportPath):], "/")
	pkgID, err := idx.selectPackageID(ctx, modID, relativePath)
	if ignoreErrNoRows(err) != nil {
		return -1, err
	}
	if pkgID > 0 {
		return pkgID, nil
	}

	pkgID, err = idx.insertPackage(ctx, modID, relativePath)
	if err != nil {
		return -1, err
	}

	return pkgID, idx.syncPartials(ctx, pkgID, pkg.ImportPath)
}

func (idx *Index) syncPartials(ctx context.Context, pkgID int64, importPath string) error {
	dlogSync.Printf("syncing partials for package %q", importPath)
	lastSlash := len(importPath)
	for lastSlash > 0 {
		lastSlash = strings.LastIndex(importPath[:lastSlash], "/")
		if _, err := idx.insertPartial(ctx, pkgID, importPath[lastSlash+1:]); err != nil {
			return fmt.Errorf("failed to insert partial: %w", err)
		}
	}
	return nil
}
