package index

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"go/build"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/benchmark"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index/schema"
	"github.com/stretchr/testify/require"
)

func init() { AddFlags(flag.CommandLine) }

type indexTest struct {
	name        string
	mods        []godoc.PackageDir
	searchTests []searchTest
}
type searchTest struct {
	paths   []string
	partial bool
	results []string
}

func (test indexTest) run(t *testing.T) {
	require := require.New(t)
	t.Helper()
	ctx := context.Background()
	pkgs, err := Load(ctx, dbMem, test.mods, loadOpts())
	require.NoError(err)
	t.Cleanup(func() {
		require.NoError(pkgs.Close())
	})
	for _, searchTest := range test.searchTests {
		searchTest.run(t, pkgs)
	}
}

func (test searchTest) run(t *testing.T, pkgs *Index) {
	t.Helper()
	ctx := context.Background()
	for _, path := range test.paths {
		name := "exact:"
		var opts []SearchOption
		if test.partial {
			name = "partial:"
			opts = append(opts, WithMatchPartials())
		}
		t.Run(name+path, func(t *testing.T) {
			results, err := pkgs.Search(ctx, path, opts...)
			require.NoError(t, err)
			require.Equal(t, test.results, importPaths(results))
		})
	}
}

func importPaths(pkgs []godoc.PackageDir) []string {
	paths := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		paths[i] = pkg.ImportPath
	}
	return paths
}

var GOROOT = build.Default.GOROOT

func stdlibCodeRoots() []godoc.PackageDir {
	return []godoc.PackageDir{
		godoc.NewPackageDir("", filepath.Join(GOROOT, "src")),
		godoc.NewPackageDir("cmd", filepath.Join(GOROOT, "src", "cmd")),
	}
}

var indexTests = []indexTest{{
	name: "stdlib",
	mods: stdlibCodeRoots(),
	searchTests: []searchTest{{
		paths:   []string{"json", "jso"},
		partial: true,
		results: []string{
			"encoding/json",
			"net/rpc/jsonrpc",
		},
	}, {
		paths:   []string{"encoding/json", "encoding/jso", "e/j"},
		partial: true,
		results: []string{"encoding/json"},
	}, {
		paths:   []string{"http"},
		partial: true,
		results: []string{"net/http", "net/http/httptest", "net/http/httptrace", "net/http/httputil", "net/http/cgi", "net/http/cookiejar", "net/http/fcgi", "net/http/internal", "net/http/pprof", "net/http/internal/ascii", "net/http/internal/testcert"},
	}, {
		paths:   []string{"http"},
		partial: false,
		results: []string{"net/http"},
	}, {
		paths:   []string{"ht"},
		partial: true,
		results: []string{"html", "net/http", "net/http/httptest", "net/http/httptrace", "net/http/httputil", "html/template", "net/http/cgi", "net/http/cookiejar", "net/http/fcgi", "net/http/internal", "net/http/pprof", "net/http/internal/ascii", "net/http/internal/testcert"},
	}, {
		paths:   []string{"a"},
		partial: true,
		results: []string{"arena", "crypto/aes", "encoding/ascii85", "encoding/asn1", "go/ast", "hash/adler32", "internal/abi", "runtime/asan", "sync/atomic", "crypto/internal/alias", "runtime/internal/atomic", "net/http/internal/ascii", "runtime/race/internal/amd64v1", "runtime/race/internal/amd64v3", "cmd/addr2line", "cmd/api", "cmd/asm", "cmd/internal/archive", "cmd/asm/internal/arch", "cmd/asm/internal/asm", "cmd/compile/internal/abi", "cmd/compile/internal/abt", "cmd/compile/internal/amd64", "cmd/compile/internal/arm", "cmd/compile/internal/arm64", "cmd/go/internal/auth", "cmd/internal/obj/arm", "cmd/internal/obj/arm64", "cmd/link/internal/amd64", "cmd/link/internal/arm", "cmd/link/internal/arm64", "archive/tar", "archive/zip", "cmd/asm/internal/flags", "cmd/asm/internal/lex"},
	}, {
		paths:   []string{"c/a"},
		partial: true,
		results: []string{"crypto/aes", "cmd/addr2line", "cmd/api", "cmd/asm", "cmd/asm/internal/arch", "cmd/asm/internal/asm", "cmd/asm/internal/flags", "cmd/asm/internal/lex"},
	}, {
		paths:   []string{"as"},
		partial: true,
		results: []string{"encoding/ascii85", "encoding/asn1", "go/ast", "runtime/asan", "net/http/internal/ascii", "cmd/asm", "cmd/asm/internal/asm", "cmd/asm/internal/arch", "cmd/asm/internal/flags", "cmd/asm/internal/lex"},
	}},
}}

func TestSearch(t *testing.T) {
	for _, test := range indexTests {
		t.Run(test.name, test.run)
	}
}

func BenchmarkSearch_partials_stdlib(b *testing.B) {
	b.Helper()
	const partial = true
	benchmarkSearch_stdlib(b, partial)
}
func BenchmarkSearch_exact_stdlib(b *testing.B) {
	b.Helper()
	const partial = false
	benchmarkSearch_stdlib(b, partial)
}

func benchmarkSearch_stdlib(b *testing.B, partial bool) {
	require := require.New(b)

	ctx := context.Background()
	codeRoots := stdlibCodeRoots()

	var searchOpts []SearchOption
	if partial {
		searchOpts = append(searchOpts, WithMatchPartials())
	}
	var matches []godoc.PackageDir
	var err error

	var pkgIdx *Index
	var randomPartial *randomPartial
	benchmark.Run(b, func() {
		pkgIdx, err = Load(ctx, dbFilePath(b), codeRoots, loadOpts())
		require.NoError(err)
		b.Cleanup(func() { require.NoError(pkgIdx.Close()) })

		randomPartial, err = pkgIdx.randomPartial()
		require.NoError(err)
		b.Cleanup(func() { require.NoError(randomPartial.Close()) })
	}, func() {
		path, err := randomPartial.randomPartial()
		require.NoError(err)
		matches, err = pkgIdx.Search(ctx, path, searchOpts...)
		require.NoError(err)
	})
	b.Log("num matches: ", len(matches))
}

func TestRandomPartialSearchPath(t *testing.T) {
	require := require.New(t)

	ctx := context.Background()
	codeRoots := stdlibCodeRoots()

	pkgIdx, err := Load(ctx, dbFilePath(t), codeRoots, loadOpts())
	require.NoError(err)
	t.Cleanup(func() { require.NoError(pkgIdx.Close()) })

	randomPartial, err := pkgIdx.randomPartial()
	require.NoError(err)
	t.Cleanup(func() { require.NoError(randomPartial.Close()) })

	paths := make(map[string]struct{})
	var duplicates int
	const total = 1000
	for i := 0; i < total; i++ {
		path, err := randomPartial.randomPartial()
		require.NoError(err)
		if _, duplicate := paths[path]; duplicate {
			// t.Log("duplicate path:", path, i)
			duplicates++
			continue
		}
		// t.Log("unique path:", path, i)
		paths[path] = struct{}{}
	}

	t.Log("duplicates:", duplicates)
	require.Less(duplicates, total/2, "too many duplicates")
}

func BenchmarkRandomPartialSearchPath(b *testing.B) {
	require := require.New(b)
	var path string
	var pkgIdx *Index
	var randomPartial *randomPartial
	var err error
	benchmark.Run(b, func() {
		ctx := context.Background()
		codeRoots := stdlibCodeRoots()
		opts := WithOptions(WithNoProgressBar())

		pkgIdx, err = Load(ctx, dbMem, codeRoots, opts)
		require.NoError(err)
		b.Cleanup(func() { require.NoError(pkgIdx.Close()) })

		randomPartial, err = pkgIdx.randomPartial()
		require.NoError(err)
		b.Cleanup(func() { require.NoError(randomPartial.Close()) })
	}, func() {
		path, err = randomPartial.randomPartial()
		require.NoError(err)
	})
	b.Log("path: ", path)
}

func (pkgIdx *Index) randomPartial() (*randomPartial, error) {
	if err := pkgIdx.waitSync(); err != nil {
		return nil, err
	}
	stmt, err := pkgIdx.db.Prepare(`
SELECT parts FROM partial ORDER BY RANDOM();
`)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	return &randomPartial{
		stmt: stmt,
		rows: rows,
	}, nil
}

type randomPartial struct {
	stmt *sql.Stmt
	rows *sql.Rows
}

func (r *randomPartial) Close() error {
	if r.stmt != nil {
		if err := r.stmt.Close(); err != nil {
			return err
		}
	}
	return nil
}
func (r *randomPartial) randomPartial() (string, error) {
	if !r.rows.Next() {
		if err := r.rows.Close(); err != nil {
			return "", err
		}
		rows, err := r.stmt.Query()
		if err != nil {
			return "", err
		}
		r.rows = rows
		if !r.rows.Next() {
			return "", fmt.Errorf("no rows")
		}
	}
	return scanImportPath(r.rows)

}
func scanImportPath(rows schema.Scanner) (string, error) {
	var path string
	return path, rows.Scan(&path)
}
