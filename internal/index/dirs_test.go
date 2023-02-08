package index

import (
	"context"
	"testing"

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

	path, err := pkgIdx.randomPartial()
	require.NoError(err)
	t.Log("filter path: ", path)
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
	dbPath := dbMem
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())
	pkgIdx, err := Load(ctx, dbPath, codeRoots, opts)
	require.NoError(err)
	dirs := NewDirs(pkgIdx)

	path, err := pkgIdx.randomPartial()
	require.NoError(err)
	t.Log("filter path: ", path)
	require.NoError(dirs.FilterExact(path))
	for {
		pkg, ok := dirs.Next()
		if !ok {
			return
		}
		t.Log("pkg: ", pkg)
	}

}

// func BenchmarkDirs(b *testing.B) {
// 	require := require.New(b)

// 	var (
// 		pkgIdx *Index
// 		pkg    godoc.PackageDir
// 		ok     bool
// 	)
// 	benchmark.Run(b, func() {
// 		ctx := context.Background()
// 		dbPath := dbMem
// 		codeRoots := stdlibCodeRoots()
// 		opts := WithOptions(WithNoProgressBar())
// 		var err error
// 		pkgIdx, err = Load(ctx, dbPath, codeRoots, opts)
// 		require.NoError(err)

// 	}, func() {
// 		path, err := pkgIdx.randomPartial()
// 		require.NoError(err)
// 		dirs := NewDirs(pkgIdx)
// 		require.NoError(dirs.FilterExact(path))
// 		for {
// 			pkg, ok = dirs.Next()
// 			if !ok || rand.Intn(100) < 80 {
// 				return
// 			}
// 		}
// 	})
// 	b.Log("pkg: ", pkg, "ok: ", ok)
// }
