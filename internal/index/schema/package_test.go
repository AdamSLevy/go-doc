package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	require := require.New(t)
	db := openTestDB(t)
	ctx := context.Background()

	mods := []Module{{
		ID:         1,
		ImportPath: "github.com/stretchr/testify",
		Dir:        "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.1",
		Class:      ClassRequired,
	}, {
		ID:         2,
		ImportPath: "github.com/muesli/reflow",
		Dir:        "/home/adam/go/pkg/mod/github.com/muesli/reflow@v0.3.0",
		Class:      ClassRequired,
	}}

	rMods, err := SyncModules(ctx, db, mods)
	require.NoError(err, "failed to insert modules")
	require.Equal(mods, rMods, "initial insert should return all modules")

	const numPackagesInFirstModule = 3
	pkgs := []Package{{
		ID:           1,
		ModuleID:     1,
		RelativePath: "",
		NumParts:     0,
	}, {
		ID:           2,
		ModuleID:     1,
		RelativePath: "assert",
		NumParts:     1,
	}, {
		ID:           3,
		ModuleID:     1,
		RelativePath: "require",
		NumParts:     1,
	}, {
		ID:           4,
		ModuleID:     2,
		RelativePath: "indent",
		NumParts:     1,
	}, {
		ID:           5,
		ModuleID:     2,
		RelativePath: "wordwrap",
		NumParts:     1,
	}, {
		ID:           6,
		ModuleID:     2,
		RelativePath: "ansi",
		NumParts:     1,
	}, {
		ID:           7,
		ModuleID:     2,
		RelativePath: "padding",
		NumParts:     1,
	}}

	rPkgs, err := SyncPackages(ctx, db, pkgs)
	require.NoError(err, "failed to sync packages")
	require.Equal(pkgs, rPkgs, "initial sync should return all packages")

	rPkgs, err = SelectAllPackages(ctx, db, rPkgs[:0])
	require.NoError(err, "failed to select packages")
	require.Equal(pkgs, rPkgs, "packages do not match")

	// sync existing packages
	rPkgs, err = SyncPackages(ctx, db, pkgs)
	require.NoError(err, "failed to sync packages")
	require.Empty(rPkgs, "syncing existing packages should return no packages")

	rPkgs, err = SelectAllPackages(ctx, db, rPkgs[:0])
	require.NoError(err, "failed to select packages")
	require.Equal(pkgs, rPkgs, "packages do not match")

	// remove a package
	rPkgs, err = SyncPackages(ctx, db, pkgs[:len(pkgs)-1])
	require.NoError(err, "failed to sync packages")
	require.Empty(rPkgs, "removing a package should return no packages")

	rPkgs, err = SelectAllPackages(ctx, db, rPkgs[:0])
	require.NoError(err, "failed to select packages")
	require.Equal(pkgs[:len(pkgs)-1], rPkgs, "packages do not match")

	// add a package back
	rPkgs, err = SyncPackages(ctx, db, pkgs)
	require.NoError(err, "failed to sync packages")
	require.Equal(pkgs[len(pkgs)-1:], rPkgs, "adding a package should return just that package")

	rPkgs, err = SelectAllPackages(ctx, db, rPkgs[:0])
	require.NoError(err, "failed to select packages")
	require.Equal(pkgs, rPkgs, "packages do not match")

	// remove packages for removed module
	rMods, err = SyncModules(ctx, db, mods[:len(mods)-1])
	require.NoError(err, "failed to insert modules")
	require.Empty(rMods, "removing a module should return no modules")

	rPkgs, err = SelectAllPackages(ctx, db, rPkgs[:0])
	require.NoError(err, "failed to select packages")
	require.Equal(pkgs[:numPackagesInFirstModule], rPkgs, "packages do not match")
}
