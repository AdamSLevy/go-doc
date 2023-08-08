package schema

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartial(t *testing.T) {
	require := require.New(t)
	db := openTestDB(t)
	ctx := context.Background()

	mods := []Module{{
		ID:         1,
		ImportPath: "github.com/stretchr/testify",
		Dir:        "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.1",
		Class:      ClassRequired,
	}}

	rMods, err := SyncModules(ctx, db, mods)
	require.NoError(err, "failed to insert modules")
	require.Equal(mods, rMods, "initial insert should return all modules")

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
	}}

	rPkgs, err := SyncPackages(ctx, db, pkgs)
	require.NoError(err, "failed to sync packages")
	require.Equal(pkgs, rPkgs, "initial sync should return all packages")

	var partials []Partial
	var partialID int64
	for _, pkg := range rPkgs {
		importPath := filepath.Join(mods[0].ImportPath, pkg.RelativePath)
		lastSlash := len(importPath)
		numParts := 0
		for lastSlash > 0 {
			partialID++
			numParts++
			lastSlash = strings.LastIndex(importPath[:lastSlash], "/")
			partials = append(partials, Partial{
				ID:        partialID,
				PackageID: pkg.ID,
				Parts:     importPath[lastSlash+1:],
				NumParts:  numParts,
			})
		}
	}

	require.NoError(SyncPartials(ctx, db, partials))

	rPartials, err := selectPartials(ctx, db, nil)
	require.NoError(err, "failed to select partials")
	require.Equal(partials, rPartials, "initial select should return all partials")

	_, err = SyncPackages(ctx, db, pkgs[:len(pkgs)-1])
	require.NoError(err, "failed to sync packages")

	rPartials, err = selectPartials(ctx, db, rPartials[:0])
	require.NoError(err, "failed to select partials")
	require.Equal(partials[:len(partials)-4], rPartials, "partials for removed package should be removed")
}
