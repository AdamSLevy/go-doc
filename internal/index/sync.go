package index

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/godoc"
)

func (idx *Index) syncCodeRoots(ctx context.Context, codeRoots []godoc.PackageDir) (retErr error) {
	commitIfNilErr, err := idx.beginTx(ctx)
	if err != nil {
		return err
	}
	defer commitIfNilErr(&retErr)

	var keep []int64
	for _, codeRoot := range codeRoots {
		modID, err := idx.syncCodeRoot(ctx, codeRoot)
		if err != nil {
			return err
		}
		keep = append(keep, modID)
	}

	const vendor = false
	if err := idx.pruneModules(ctx, vendor, keep); err != nil {
		return err
	}

	return idx.upsertSync(ctx)
}
func (idx *Index) beginTx(ctx context.Context) (commitIfNilErr func(*error), _ error) {
	tx, err := idx.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	idx.tx = tx
	return func(retErr *error) {
		idx.tx = nil
		if *retErr != nil {
			tx.Rollback()
			return
		}
		*retErr = tx.Commit()
	}, nil
}

func (idx *Index) syncCodeRoot(ctx context.Context, root godoc.PackageDir) (modID int64, _ error) {
	class, vendor := parseClassVendor(root.ImportPath, root.Dir)
	if vendor {
		// ImportPath is empty for vendor directories, so we use the
		// Dir instead so as not to conflict with the stdlib, which
		// uses the empty import path.
		//
		// The root vendor module is a place holder, so its import path
		// will never be used to form a package path.
		root.ImportPath = root.Dir
	}

	modID, needsSync, err := idx.upsertModule(ctx, root, class, vendor)
	if err != nil {
		return -1, err
	}

	if !needsSync {
		return modID, nil
	}

	if vendor {
		return modID, idx.syncVendoredModules(ctx, root)
	}

	return modID, idx.syncModulePackages(ctx, modID, root)
}
func parseClassVendor(importPath, dir string) (class, bool) {
	if isVendor(dir) {
		return classRequired, true
	}
	switch importPath {
	case "", "cmd":
		return classStdlib, false
	}
	if _, hasVersion := parseVersion(dir); hasVersion {
		return classRequired, false
	}
	return classLocal, false
}
func parseVersion(dir string) (string, bool) {
	_, version, found := strings.Cut(filepath.Base(dir), "@")
	return version, found
}
func isVendor(dir string) bool { return filepath.Base(dir) == "vendor" }

func (idx *Index) upsertModule(ctx context.Context, root godoc.PackageDir, class class, vendor bool) (modID int64, needsSync bool, _ error) {
	mod, err := idx.loadModule(ctx, root.ImportPath)
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
	root.Dir = filepath.Clean(root.Dir) // because filepath.Join will do it anyway

	// this is the queue of directories to examine in this pass.
	this := []godoc.PackageDir{}
	// next is the queue of directories to examine in the next pass.
	next := []godoc.PackageDir{root}

	var keep []int64
	for len(next) > 0 {
		dlog.Printf("descending")
		this, next = next, this[0:0]
		for _, pkg := range this {
			dlog.Printf("walking %q", pkg)
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
						dlog.Printf("%q has go files", pkg)
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
				// When in a module, ignore vendor directories and stop at module boundaries.
				if vendor := false; vendor {
					if name == "vendor" {
						continue
					}
					if fi, err := os.Stat(filepath.Join(pkg.Dir, name, "go.mod")); err == nil && !fi.IsDir() {
						continue
					}
				}
				// Remember this (fully qualified) directory for the next pass.
				subPkg := godoc.NewPackageDir(
					path.Join(pkg.ImportPath, name),
					filepath.Join(pkg.Dir, name),
				)
				dlog.Printf("queuing %q", subPkg.ImportPath)
				next = append(next, subPkg)
			}
		}
	}

	return idx.prunePackages(ctx, modID, keep)
}

func (idx *Index) syncPackage(ctx context.Context, modID int64, root, pkg godoc.PackageDir) (int64, error) {
	relativePath := strings.TrimPrefix(pkg.ImportPath[len(root.ImportPath):], "/")
	pkgID, err := idx.getPackageID(ctx, modID, relativePath)
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
	lastSlash := len(importPath)
	for lastSlash > 0 {
		lastSlash = strings.LastIndex(importPath[:lastSlash], "/")
		if _, err := idx.insertPartial(ctx, pkgID, importPath[lastSlash+1:]); err != nil {
			return fmt.Errorf("failed to insert partial: %w", err)
		}
	}
	return nil
}

func (idx *Index) syncVendoredModules(ctx context.Context, vendorRoot godoc.PackageDir) error {
	if idx.vendorUnchanged(vendorRoot) {
		return nil
	}
	const modulesTxt = "modules.txt"
	modTxtPath := filepath.Join(vendorRoot.Dir, modulesTxt)
	f, err := os.Open(modTxtPath)
	if err != nil {
		log.Printf("failed to open %s: %v", modTxtPath, err)
		return nil
	}
	defer f.Close()

	return idx.syncVendoredModulesTxtFile(ctx, vendorRoot, f)
}
func (idx *Index) vendorUnchanged(vendor godoc.PackageDir) bool {
	info, err := os.Stat(vendor.Dir)
	if err != nil {
		log.Printf("failed to stat %s: %v", vendor.Dir, err)
		return true
	}
	return idx.UpdatedAt.After(info.ModTime())
}
func (idx *Index) syncVendoredModulesTxtFile(ctx context.Context, vendorRoot godoc.PackageDir, data io.Reader) error {
	const vendor = true
	var (
		err              error
		modID            int64
		modRoot          godoc.PackageDir
		modKeep, pkgKeep []int64
	)
	lines := bufio.NewScanner(data)
	for lines.Scan() && ctx.Err() == nil {
		modImportPath, _, pkgImportPath := parseModuleTxtLine(lines.Text())
		if modImportPath != "" {
			if modID > 0 {
				if err := idx.prunePackages(ctx, modID, pkgKeep); err != nil {
					return err
				}
				pkgKeep = pkgKeep[:0]
			}
			modRoot = godoc.NewPackageDir(
				modImportPath,
				filepath.Join(vendorRoot.Dir, modImportPath),
			)
			modID, _, err = idx.upsertModule(ctx, modRoot, classRequired, vendor)
			if err != nil {
				return err
			}
			modKeep = append(modKeep, modID)
			continue
		}
		if pkgImportPath != "" && modID > 0 {
			pkgID, err := idx.syncPackage(ctx, modID, modRoot, godoc.NewPackageDir(pkgImportPath, ""))
			if err != nil {
				return err
			}
			pkgKeep = append(pkgKeep, pkgID)
		}
	}
	if modID > 0 && len(pkgKeep) > 0 {
		if err := idx.prunePackages(ctx, modID, pkgKeep); err != nil {
			return err
		}
	}

	return idx.pruneModules(ctx, vendor, modKeep)
}
func parseModuleTxtLine(line string) (modImportPath, modVersion, pkgImportPath string) {
	defer func() {
		dlog.Printf("parseModuleTxtLine(%q) (%q, %q, %q)",
			line, modImportPath, modVersion, pkgImportPath)
	}()
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	switch fields[0] {
	case "#":
		// module
		if len(fields) < 3 {
			return
		}
		modImportPath, modVersion = fields[1], fields[2]
		if !strings.HasPrefix(modVersion, "v") {
			modVersion = ""
		}
	case "##":
		// ignore
	default:
		pkgImportPath = fields[0]
	}
	return
}
