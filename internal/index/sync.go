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
	"aslevy.com/go-doc/internal/index/schema"
)

var dlogSync = dlog.Child("sync")

func (idx *Index) needsSync(ctx context.Context) (bool, error) {
	switch idx.options.mode {
	case ModeOff, ModeSkipSync:
		return false, nil
	case ModeForceSync:
		return true, nil
	}
	var err error
	idx.Metadata, err = schema.SelectMetadata(ctx, idx.db)
	if ignoreErrNoRows(err) != nil {
		return false, err
	}

	if idx.Metadata.BuildRevision != schema.BuildRevision ||
		idx.Metadata.GoVersion != schema.GoVersion {
		return true, nil
	}

	dlogSync.Printf("created at: %v", idx.CreatedAt.Local())
	dlogSync.Printf("updated at: %v", idx.UpdatedAt.Local())
	return time.Since(idx.UpdatedAt) > idx.options.resyncInterval, nil
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

	dlogSync.Println("syncing code roots...")
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer schema.CommitOrRollback(tx, &retErr)

	pb := newProgressBar(idx.options, len(codeRoots)+1, "syncing code roots")
	defer pb.Finish()

	mods := make([]schema.Module, len(codeRoots))
	for i, codeRoot := range codeRoots {
		mods[i] = codeRootToModule(codeRoot)
	}

	updatedMods, err := schema.SyncModules(ctx, tx, mods)
	if err != nil {
		return err
	}

	var pkgs []schema.Package
	var partials []schema.Partial
	for _, mod := range updatedMods {
		modPkgs, err := idx.syncModulePackages(ctx, mod)
		if err != nil {
			return err
		}
		for _, pkg := range newPkgs {

		}
		pkgs = append(pkgs, modPkgs...)

	}

	newPkgs, err := schema.SyncPackages(ctx, tx, pkgs)
	if err != nil {
		return err
	}

	err := schema.SyncPartials(ctx, tx, partials)
	if err != nil {
		return err
	}

	pb.Add(1)

	return schema.UpsertMetadata(ctx, tx)
}
func codeRootToModule(codeRoot godoc.PackageDir) schema.Module {
	class, vendor := parseClassVendor(codeRoot)
	return schema.Module{
		ImportPath: codeRoot.ImportPath,
		Dir:        codeRoot.Dir,
		Class:      class,
		Vendor:     vendor,
	}
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

	if !needsSync && idx.options.mode != ModeForceSync {
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

	if mod.ID < 1 {
		mod.ID, err = idx.insertModule(ctx, root, class, vendor)
		if err != nil {
			return -1, false, err
		}
	} else {
		if err := idx.updateModule(ctx, mod.ID, root, class, vendor); err != nil {
			return -1, false, err
		}
	}
	return mod.ID, true, nil
}

func (idx *Index) syncModulePackages(ctx context.Context, root schema.Module) ([]schema.Package, error) {
	dlogSync.Printf("syncing module packages for %q in %q", root.ImportPath, root.Dir)
	root.Dir = filepath.Clean(root.Dir) // because filepath.Join will do it anyway

	pkgs := make([]schema.Package, 0, 100)

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
						pkgs = append(pkgs, schema.Package{
							ModuleID:     root.ID,
							RelativePath: relativePath,
						})
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

	return pkgs, ctx.Err()
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
