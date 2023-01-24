package index

import (
	"go/build"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aslevy.com/go-doc/internal/godoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type indexTest struct {
	name        string
	mods        []godoc.PackageDir
	searchTests []searchTest
}
type searchTest struct {
	name    string
	paths   []string
	exact   bool
	results []string
}

func (test indexTest) run(t *testing.T) {
	t.Helper()
	pkgs := New(test.mods)
	for _, searchTest := range test.searchTests {
		t.Run(searchTest.name, func(t *testing.T) { searchTest.run(t, pkgs) })
	}
}

func (test searchTest) run(t *testing.T, pkgs *Packages) {
	t.Helper()
	for _, path := range test.paths {
		t.Run("path/"+path, func(t *testing.T) {
			results := pkgs.Search(path, test.exact)
			require.Len(t, results, len(test.results))
			for i, result := range results {
				assert.Equal(t, test.results[i], result.ImportPath)
				assert.NotEmpty(t, result.Dir)
			}
		})
	}
}

var GOROOT = build.Default.GOROOT

var indexTests = []indexTest{{
	name: "stdlib",
	mods: []godoc.PackageDir{
		{"", filepath.Join(GOROOT, "src")},
		{"cmd", filepath.Join(GOROOT, "src", "cmd")},
	},
	searchTests: []searchTest{{
		name:  "json",
		paths: []string{"json", "jso"},
		results: []string{
			"encoding/json",
			"net/rpc/jsonrpc",
		},
	}, {
		name:    "encoding/json",
		paths:   []string{"encoding/json", "encoding/jso", "e/j"},
		results: []string{"encoding/json"},
	}, {
		name:    "http",
		paths:   []string{"http"},
		results: []string{"net/http", "net/http/httptrace"},
	}, {
		name:    "http",
		paths:   []string{"http"},
		exact:   true,
		results: []string{"net/http"},
	}, {
		name:    "ht",
		paths:   []string{"ht"},
		results: []string{"html", "html/template", "net/http", "net/http/httptrace"},
	}, {
		name:  "a",
		paths: []string{"a"},
		results: []string{
			"archive/tar",
			"archive/zip",
			"crypto/aes",
			"encoding/ascii85",
			"encoding/asn1",
			"go/ast",
			"hash/adler32",
			"internal/abi",
			"runtime/asan",
			"sync/atomic",
			"cmd/addr2line",
			"cmd/api",
			"cmd/asm",
			"cmd/internal/archive",
			"cmd/asm/internal/arch",
			"cmd/internal/obj/arm64",
			"cmd/link/internal/arm64",
		},
	}, {
		name:  "c/a",
		paths: []string{"c/a"},
		results: []string{
			"crypto/aes",
			"cmd/addr2line",
			"cmd/api",
			"cmd/asm",
			"cmd/asm/internal/arch",
		},
	}, {
		name:  "as",
		paths: []string{"as"},
		results: []string{
			"encoding/ascii85",
			"encoding/asn1",
			"go/ast",
			"runtime/asan",
			"cmd/asm",
			"cmd/asm/internal/arch",
		},
	}},
}}

func TestSearch(t *testing.T) {
	for _, test := range indexTests {
		t.Run(test.name, func(t *testing.T) { test.run(t) })
	}
}

func BenchmarkSearch_stdlib(b *testing.B) {
	var pkgIdx *Packages
	var matches []godoc.PackageDir
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	var randomPartialSearchPath func() string
	exact := false
	runBenchmark(b, func() {
		pkgIdx = New(codeRoots, WithNoProgressBar())
		randomPartialSearchPath = newRandomPartialSearchPathFunc(pkgIdx)
	}, func() {
		path := randomPartialSearchPath()
		matches = pkgIdx.Search(path, exact)
	})
	b.Log("num matches: ", len(matches))
}

func TestRandomPartialSearchPath(t *testing.T) {
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	pkgIdx := New(codeRoots, WithNoProgressBar())

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
	var pkgIdx *Packages
	codeRoots := []godoc.PackageDir{
		{"", filepath.Join(build.Default.GOROOT, "src")},
		{"cmd", filepath.Join(build.Default.GOROOT, "src", "cmd")},
	}
	var randomPartialSearchPath func() string
	runBenchmark(b, func() {
		pkgIdx = New(codeRoots, WithNoProgressBar())
		randomPartialSearchPath = newRandomPartialSearchPathFunc(pkgIdx)
	}, func() {
		path = randomPartialSearchPath()
	})
	b.Log("path: ", path)
}

func init() { rand.Seed(time.Now().UnixNano()) }
func newRandomPartialSearchPathFunc(pkgIdx *Packages) func() string {
	// build list of all package import paths split into parts.
	var pkgs [][]string
	for _, mod := range pkgIdx.modules {
		var modParts []string
		if mod.ImportPath != "" {
			modParts = strings.Split(mod.ImportPath, "/")
		}
		for _, pkg := range mod.Packages {
			pkgParts := append(modParts, pkg.ImportPathParts[1:]...)
			pkgs = append(pkgs, pkgParts)
		}
	}
	return func() string {
		// random package
		pkg := pkgs[rand.Intn(len(pkgs))]

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
