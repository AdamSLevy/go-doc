package modpkg

import (
	"context"
	"fmt"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

func (modPkg *ModPkg) FilterExact(importPath string) error {
	return modPkg.filter(context.TODO(), importPath, true)
}
func (modPkg *ModPkg) FilterPartial(importPath string) error {
	return modPkg.filter(context.TODO(), importPath, false)
}
func (modPkg *ModPkg) filter(ctx context.Context, importPath string, exact bool) error {
	if modPkg.search == importPath && modPkg.exact == exact {
		return nil
	}

	if modPkg.g != nil {
		if err := modPkg.g.Wait(); err != nil {
			return fmt.Errorf("failed to wait for sync: %w", err)
		}
		modPkg.g = nil
	} else if modPkg.rows != nil {
		if err := modPkg.rows.Close(); err != nil {
			return fmt.Errorf("failed to close previously open package search query: %w", err)
		}
		modPkg.offset = 0
		modPkg.results = modPkg.results[:0]
	}

	modPkg.exact = exact
	modPkg.search = importPath

	parts := strings.Split(importPath, "/")
	var err error
	modPkg.rows, err = modPkg.db.SelectPackagesByPartsRows(ctx, modPkg.exact, parts)
	return err
}

func (modPkg *ModPkg) Reset() { modPkg.offset = 0 }
func (modPkg *ModPkg) Next() (godoc.PackageDir, bool) {
	if modPkg.offset < len(modPkg.results) {
		// We have a result in memory.
		next := modPkg.offset
		modPkg.offset++
		return modPkg.results[next], true
	}
	// We don't have a result in memory.

	// The database query was already closed so we have no more results.
	if modPkg.rows == nil {
		return godoc.PackageDir{}, false
	}

	// Check if there is a next row.
	if !modPkg.rows.Next() {
		// See if there is an error or if we're just at the end of the
		// results.
		if err := modPkg.rows.Err(); err != nil {
			dlog.Printf("failed to read next package from db: %v", err)
		}
		// Close the database query since we're done with it.
		if err := modPkg.rows.Close(); err != nil {
			dlog.Printf("failed to close rows for package query: %v", err)
		}
		// Set the database query to nil so we know we're done with it.
		modPkg.rows = nil
		// Return that there are no more results.
		return godoc.PackageDir{}, false
	}

	// Read the next row from the database query.
	var pkg godoc.PackageDir
	if err := modPkg.rows.Scan(&pkg.ImportPath, &pkg.Dir); err != nil {
		dlog.Printf("failed to scan next package from db: %v", err)
		return godoc.PackageDir{}, false
	}
	modPkg.results = append(modPkg.results, pkg)
	modPkg.offset++
	return pkg, true
}
