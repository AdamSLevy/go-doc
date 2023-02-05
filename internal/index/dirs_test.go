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
	const partial = true
	ctx := context.Background()
	dbPath := dbMem
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())
	pkgIdx, err := Load(ctx, dbPath, codeRoots, opts)
	require.NoError(err)
	dirs := NewDirs(pkgIdx)

	randomPartialSearchPath := newRandomPartialSearchPathFunc(pkgIdx, partial)
	path := randomPartialSearchPath()
	t.Log("filter path: ", path)
	require.NoError(dirs.Filter(path, partial))
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
	const partial = false
	ctx := context.Background()
	dbPath := dbMem
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())
	pkgIdx, err := Load(ctx, dbPath, codeRoots, opts)
	require.NoError(err)
	dirs := NewDirs(pkgIdx)

	randomPartialSearchPath := newRandomPartialSearchPathFunc(pkgIdx, partial)
	path := randomPartialSearchPath()
	t.Log("filter path: ", path)
	require.NoError(dirs.Filter(path, partial))
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

	const partial = false
	var (
		dirs                    godoc.Dirs
		pkg                     godoc.PackageDir
		ok                      bool
		randomPartialSearchPath func() string
	)
	benchmark.Run(b, func() {
		ctx := context.Background()
		dbPath := dbMem
		codeRoots := stdlibCodeRoots()
		opts := WithOptions(WithNoProgressBar())
		pkgIdx, err := Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		dirs = NewDirs(pkgIdx)

		randomPartialSearchPath = newRandomPartialSearchPathFunc(pkgIdx, partial)
	}, func() {
		require.NoError(dirs.Filter(randomPartialSearchPath(), partial))
		for {
			pkg, ok = dirs.Next()
			if !ok || rand.Intn(100) < 80 {
				return
			}
		}
	})
	b.Log("pkg: ", pkg, "ok: ", ok)
}
