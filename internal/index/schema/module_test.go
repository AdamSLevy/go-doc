package schema

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModule(t *testing.T) {
	require := require.New(t)
	db := openTestDB(t)
	ctx := context.Background()

	mods := []Module{{
		ID:         1,
		ImportPath: "github.com/stretchr/testify/require",
		Dir:        "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.7.0/require",
		Class:      ClassRequired,
	}, {
		ID:         2,
		ImportPath: "github.com/stretchr/testify/assert",
		Dir:        "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.7.0/assert",
		Class:      ClassRequired,
	}}

	// initial insert
	rMods, err := SyncModules(ctx, db, mods)
	require.NoError(err, "failed to insert modules")
	require.Equal(mods, rMods, "initial insert should return all modules")

	rMods, err = selectModules(ctx, db, rMods[:0])
	require.NoError(err, "failed to select modules")
	require.Equal(mods, rMods, "modules do not match")

	// insert existing modules
	rMods, err = SyncModules(ctx, db, mods)
	require.NoError(err, "failed to insert modules")
	require.Empty(rMods, "inserting existing modules should return no modules")

	rMods, err = selectModules(ctx, db, rMods[:0])
	require.NoError(err, "failed to select modules")
	require.Equal(mods, rMods, "modules do not match")

	// modify the dir of one module and add a new module
	mods[1].Dir = "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.7.1/assert"
	mods = append(mods, Module{
		ID:         3,
		ImportPath: "modernc.org/sqlite",
		Dir:        "/home/adam/go/pkg/mod/modernc.org/sqlite@v0.0.0-20210102203006-7e8e4e7d4f0e",
		Class:      ClassRequired,
	})

	rMods, err = SyncModules(ctx, db, mods)
	require.NoError(err, "failed to insert modules")
	require.Equal(mods[1:3], rMods, "we should only get the new and modified modules")

	rMods, err = selectModules(ctx, db, rMods[:0])
	require.NoError(err, "failed to select modules")
	require.Equal(mods, rMods, "modules do not match")

	// modify the dir of one module and remove another
	mods[2].Dir = "/home/adam/go/pkg/mod/modernc.org/sqlite@v1.24.0"
	rMods, err = SyncModules(ctx, db, mods[1:])
	require.NoError(err, "failed to insert modules")
	require.Equal(mods[2:3], rMods, "we should only get the modified module")

	rMods, err = selectModules(ctx, db, rMods[:0])
	require.NoError(err, "failed to select modules")
	require.Equal(mods[1:], rMods, "modules do not match")
}
