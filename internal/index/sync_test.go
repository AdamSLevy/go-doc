package index

import (
	"context"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
	"github.com/stretchr/testify/require"
)

const (
	dbMem  = ":memory:"
	dbFile = "file:"
	dbName = "index.sqlite3"
)

func dbFilePath(dir string) string { return dbFile + filepath.Join(dir, dbName) }

func BenchmarkLoadSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())

	var idx *Index
	var err error
	benchmark.Run(b, nil, func() {
		idx, err = Load(ctx, dbFilePath(b.TempDir()), codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})

	b.Logf("index sync %+v", idx.sync)
}

func BenchmarkLoadSync_InMemory_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbMem
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())

	var idx *Index
	var err error
	benchmark.Run(b, nil, func() {
		idx, err = Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})
	b.Logf("index sync %+v", idx.sync)
}

func BenchmarkLoadReSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b.TempDir())
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar(), WithResyncInterval(0))

	var idx *Index
	var err error
	benchmark.Run(b, func() {
		// sync initially prior to running the benchmark
		idx, err = Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	}, func() {
		idx, err = Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})
	b.Logf("index sync %+v", idx.sync)
}

func BenchmarkLoadNoSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b.TempDir())
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(WithNoProgressBar())

	var idx *Index
	var err error
	benchmark.Run(b, func() {
		// sync initially prior to running the benchmark
		idx, err = Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	}, func() {
		idx, err = Load(ctx, dbPath, codeRoots, opts)
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})
	b.Logf("index sync %+v", idx.sync)
}
