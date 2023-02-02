package index

import (
	"context"
	"flag"
	"go/build"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aslevy.com/go-doc/internal/benchmark"
	"aslevy.com/go-doc/internal/godoc"
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
	t.Helper()
	ctx := context.Background()
	const dbPath = ":memory:"
	pkgs, err := Load(ctx, dbPath, test.mods, WithNoProgressBar())
	require.NoError(t, err)
	for _, searchTest := range test.searchTests {
		searchTest.run(t, pkgs)
	}
}

func (test searchTest) run(t *testing.T, pkgs *Index) {
	t.Helper()
	ctx := context.Background()
	for _, path := range test.paths {
		name := "exact:"
		if test.partial {
			name = "partial:"
		}
		t.Run(name+path, func(t *testing.T) {
			results, err := pkgs.Search(ctx, path, test.partial)
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

var indexTests = []indexTest{{
	name: "stdlib",
	mods: []godoc.PackageDir{
		{"", filepath.Join(GOROOT, "src")},
		{"cmd", filepath.Join(GOROOT, "src", "cmd")},
	},
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
		results: []string{"crypto/aes", "encoding/ascii85", "encoding/asn1", "go/ast", "hash/adler32", "internal/abi", "runtime/asan", "sync/atomic", "runtime/internal/atomic", "net/http/internal/ascii", "cmd/addr2line", "cmd/api", "cmd/asm", "cmd/internal/archive", "cmd/asm/internal/arch", "cmd/asm/internal/asm", "cmd/compile/internal/abi", "cmd/compile/internal/abt", "cmd/compile/internal/amd64", "cmd/compile/internal/arm", "cmd/compile/internal/arm64", "cmd/go/internal/auth", "cmd/internal/obj/arm", "cmd/internal/obj/arm64", "cmd/link/internal/amd64", "cmd/link/internal/arm", "cmd/link/internal/arm64", "archive/tar", "archive/zip", "cmd/asm/internal/flags", "cmd/asm/internal/lex"},
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

func BenchmarkSearch_stdlib(b *testing.B) {
	var pkgIdx *Index
	var matches []godoc.PackageDir
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	var randomPartialSearchPath func() string
	var err error
	ctx := context.Background()
	dbPath := ":memory:"
	exact := false
	benchmark.Run(b, func() {
		pkgIdx, err = Load(ctx, dbPath, codeRoots, WithNoProgressBar())
		require.NoError(b, err)
		randomPartialSearchPath = newRandomPartialSearchPathFunc(pkgIdx)
	}, func() {
		path := randomPartialSearchPath()
		matches, err = pkgIdx.Search(ctx, path, exact)
		require.NoError(b, err)
	})
	b.Log("num matches: ", len(matches))
}

func TestRandomPartialSearchPath(t *testing.T) {
	ctx := context.Background()
	dbPath := ":memory:"
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	pkgIdx, err := Load(ctx, dbPath, codeRoots, WithNoProgressBar())
	require.NoError(t, err)

	randomPartialSearchPath := newRandomPartialSearchPathFunc(pkgIdx)

	paths := make(map[string]struct{})
	var duplicates int
	const total = 1000
	for i := 0; i < total; i++ {
		path := randomPartialSearchPath()
		if _, duplicate := paths[path]; duplicate {
			// t.Log("duplicate path:", path, i)
			duplicates++
			continue
		}
		// t.Log("unique path:", path, i)
		paths[path] = struct{}{}
	}

	t.Log("duplicates:", duplicates)
	require.Less(t, duplicates, total/2, "too many duplicates")
}

func BenchmarkRandomPartialSearchPath(b *testing.B) {
	var path string
	var pkgIdx *Index
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	ctx := context.Background()
	dbPath := ":memory:"
	var err error
	var randomPartialSearchPath func() string
	benchmark.Run(b, func() {
		pkgIdx, err = Load(ctx, dbPath, codeRoots, WithNoProgressBar())
		require.NoError(b, err)
		randomPartialSearchPath = newRandomPartialSearchPathFunc(pkgIdx)
	}, func() {
		path = randomPartialSearchPath()
	})
	b.Log("path: ", path)
}

func init() { rand.Seed(time.Now().UnixNano()) }
func newRandomPartialSearchPathFunc(pkgIdx *Index) func() string {
	pkgs, err := pkgIdx.Search(context.Background(), "", true)
	if err != nil {
		panic(err)
	}
	pkgParts := make([][]string, len(pkgs))
	for i, pkg := range pkgs {
		pkgParts[i] = strings.Split(pkg.ImportPath, "/")
	}
	return func() string {
		pkg := pkgParts[rand.Intn(len(pkgs))]

		// random selection of one or more parts
		first := rand.Intn(len(pkg))
		last := rand.Intn(len(pkg))
		if first > last {
			first, last = last, first
		}

		parts := pkg[first : last+1]

		// random truncation of each part
		var path string
		minLen := 3
		for _, part := range parts {
			partLen := minLen
			if partLen > len(part) {
				partLen = len(part)
			} else if partLen < len(part) {
				partLen += rand.Intn(len(part) - partLen)
			}
			path += part[:partLen] + "/"
		}
		return path[:len(path)-1] // omit trailing slash
	}
}
