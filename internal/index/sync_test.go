package index

import (
	"context"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
	"github.com/stretchr/testify/require"
)

const dbMem = ":memory:"

func dbFilePath(tb testing.TB) string {
	const (
		dbFile = "file:"
		dbName = "index.sqlite3"
	)
	path := dbFile + filepath.Join(tb.TempDir(), dbName)
	// tb.Log("db path: ", path)
	return path
}

func loadOpts() Option { return WithOptions(WithNoProgressBar(), WithResyncInterval(0)) }

// BenchmarkLoadSync_stdlib benchmarks the time it takes to sync an index of
// the stdlib from scratch and write it to the filesystem.
func BenchmarkLoadSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b)
	codeRoots := stdlibCodeRoots()

	var idx *Index
	var err error
	benchmark.Run(b, nil, func() {
		idx, err = Load(ctx, dbPath, codeRoots, loadOpts())
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})

	b.Logf("index sync %+v", idx.Metadata)
}

// BenchmarkLoadSync_InMemory_stdlib is like BenchmarkLoadSync_stdlib, but uses
// an in memory database instead of the filesystem.
func BenchmarkLoadSync_InMemory_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	codeRoots := stdlibCodeRoots()

	var idx *Index
	var err error
	benchmark.Run(b, nil, func() {
		idx, err = Load(ctx, dbMem, codeRoots, loadOpts())
		require.NoError(err)
		require.NoError(idx.waitSync())
		require.NoError(idx.Close())
	})
	b.Logf("index sync %+v", idx.Metadata)
}

// BenchmarkLoadReSync_stdlib benchmarks the time it takes to re-sync an
// existing index of the stdlib when it has not changed.
func BenchmarkLoadReSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b)
	codeRoots := stdlibCodeRoots()
	opts := loadOpts()

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
	b.Logf("index sync %+v", idx.Metadata)
}

// BenchmarkLoadForceSync_stdlib benchmarks the time it takes to re-sync an
// existing index of the stdlib when it has not changed.
func BenchmarkLoadForceSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b)
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(loadOpts(), WithForceSync())

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
	b.Logf("index sync %+v", idx.Metadata)
}

// BenchmarkLoadSkipSync_stdlib benchmarks the time it takes to load an
// existing index of the stdlib without syncing.
func BenchmarkLoadSkipSync_stdlib(b *testing.B) {
	require := require.New(b)

	ctx := context.Background()
	dbPath := dbFilePath(b)
	codeRoots := stdlibCodeRoots()
	opts := WithOptions(loadOpts(), WithSkipSync())

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
	b.Logf("index sync %+v", idx.Metadata)
}
