package index

import (
	"context"
	"math/rand"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
	"aslevy.com/go-doc/internal/godoc"
	"github.com/stretchr/testify/require"
)

func TestDirs_partial(t *testing.T) {
	require := require.New(t)
	ctx := context.Background()
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())
	pkgIdx, err := Load(ctx, dbFilePath(t), codeRoots, opts)
	require.NoError(err)
	t.Cleanup(func() { require.NoError(pkgIdx.Close()) })

	path, err := pkgIdx.randomPartial()
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
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())
	pkgIdx, err := Load(ctx, dbFilePath(t), codeRoots, opts)
	require.NoError(err)
	t.Cleanup(func() { require.NoError(pkgIdx.Close()) })

	path, err := pkgIdx.randomPartial()
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

func BenchmarkDirs(b *testing.B) {
	require := require.New(b)
	var (
		pkgIdx *Index
		pkg    godoc.PackageDir
		ok     bool
		i      int
	)
	benchmark.Run(b, func() {
		ctx := context.Background()
		codeRoots := stdlibCodeRoots()
		opts := WithOptions(WithNoProgressBar())
		var err error
		pkgIdx, err = Load(ctx, dbFilePath(b), codeRoots, opts)
		require.NoError(err)
		b.Cleanup(func() { require.NoError(pkgIdx.Close()) })
	}, func() {
		i++
		path, err := pkgIdx.randomPartial()
		require.NoError(err)

		dirs := NewDirs(pkgIdx)
		require.NoError(dirs.FilterExact(path))
		for {
			pkg, ok = dirs.Next()
			if !ok || rand.Intn(100) < 80 {
				return
			}
		}
	})
	b.Log("pkg: ", pkg, "ok: ", ok)
}
