package index

import (
	"go/build"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

type indexTest struct {
	name        string
	mods        []Module
	searchTests []searchTest
}
type searchTest struct {
	name    string
	paths   []string
	opts    []SearchOption
	results []string
}

func (test indexTest) run(t *testing.T) {
	t.Helper()
	pkgs := New(test.mods...)
	for _, searchTest := range test.searchTests {
		t.Run(searchTest.name, func(t *testing.T) { searchTest.run(t, pkgs) })
	}
}

func (test searchTest) run(t *testing.T, pkgs *Packages) {
	t.Helper()
	for _, path := range test.paths {
		t.Run("path/"+path, func(t *testing.T) {
			results := pkgs.Search(path, test.opts...)
			assert.Equal(t, test.results, results)
		})
	}
}

var GOROOT = build.Default.GOROOT

var indexTests = []indexTest{{
	name: "stdlib",
	mods: []Module{
		NewModule("", filepath.Join(GOROOT, "src")),
		NewModule("cmd", filepath.Join(GOROOT, "src", "cmd")),
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
		opts:    []SearchOption{SearchExact()},
		results: []string{"net/http"},
	}, {
		name:    "ht",
		paths:   []string{"ht"},
		results: []string{"html", "net/http", "net/http/httptrace"},
	}, {
		name:  "a",
		paths: []string{"a"},
		results: []string{
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
		},
	}},
}}

func TestSearch(t *testing.T) {
	for _, test := range indexTests {
		t.Run(test.name, func(t *testing.T) { test.run(t) })
	}
}
