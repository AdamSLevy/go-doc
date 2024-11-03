package db

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchema(t *testing.T) {
	require := require.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tmpDir := t.TempDir()
	db, err := Open(ctx, tmpDir)
	require.NoError(err, "Open")
	require.NotNil(db, "Open")
	t.Cleanup(func() { require.NoError(db.Close(), "DB.Close") })

	populateDB(t, ctx, db,
		NewModulePackages("github.com/good-coder/foobar", "v1.0.0",
			"", "foo", "bar", "baz", "foo/baz", "foo/bir/baz"),
		NewModulePackages("github.com/bad-coder/foobar", "v1.0.1",
			"", "foo", "bur", "buz", "foo/buz", "foo/bur/buz"),
	)
	// initial sync
	// add modules and packages
	//
	// check tables

}

type ModulePackages struct {
	ImportPath string
	Version    string
	Packages   []string
}

func NewModulePackages(importPath, version string, packages ...string) ModulePackages {
	for i, pkg := range packages {
		pkg = strings.TrimPrefix(pkg, importPath)
		pkg = strings.TrimPrefix(pkg, "/")
		packages[i] = pkg
	}
	return ModulePackages{
		ImportPath: importPath,
		Version:    version,
		Packages:   packages,
	}
}

func populateDB(t *testing.T, ctx context.Context, db *DB, modPkgs ...ModulePackages) {
	require := require.New(t)
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	sync, err := db.Sync(ctx)
	require.NoError(err, "DB.StartSyncIfNeeded")
	require.NotNil(sync, "DB.StartSyncIfNeeded")

	for _, mp := range modPkgs {
		mod, err := sync.AddModule(ctx, mp.ImportPath, mp.Version)
		require.NoError(err, "Sync.AddModule")
		require.NotNil(mod, "Sync.AddModule")

		for _, pkg := range mp.Packages {
			require.NoError(sync.AddPackage(ctx, mod, pkg), "Sync.AddPackage")
		}
	}

	require.NoError(sync.Finish(ctx), "Sync.Finish")
}
