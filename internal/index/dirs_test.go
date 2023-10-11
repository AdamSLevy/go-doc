package index

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirs_partial(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	pkgIdx, err := Load(ctx, dbFilePath(t), "", "", "", stdlibCodeRoots(), loadOpts())
	require.NoError(err)
	t.Cleanup(func() { require.NoError(pkgIdx.Close()) })

	randomPartial, err := pkgIdx.randomPartial()
	require.NoError(err)
	t.Cleanup(func() { require.NoError(randomPartial.Close()) })

	path, err := randomPartial.randomPartial()
	require.NoError(err)
	t.Log("filter path: ", path)

	dirs := NewDirs(pkgIdx)
	require.NoError(dirs.FilterPartial(path))
	for {
		pkg, ok := dirs.Next()
		if !ok {
			return
		}
		t.Log("pkg: ", pkg)
	}
}

func TestDirs_exact(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	pkgIdx, err := Load(ctx, dbFilePath(t), "", "", "", stdlibCodeRoots(), loadOpts())
	require.NoError(err)
	t.Cleanup(func() { require.NoError(pkgIdx.Close()) })

	randomPartial, err := pkgIdx.randomPartial()
	require.NoError(err)
	t.Cleanup(func() { require.NoError(randomPartial.Close()) })

	path, err := randomPartial.randomPartial()
	require.NoError(err)
	t.Log("filter path: ", path)

	dirs := NewDirs(pkgIdx)
	require.NoError(dirs.FilterExact(path))
	for {
		pkg, ok := dirs.Next()
		if !ok {
			return
		}
		t.Log("pkg: ", pkg)
	}
}
