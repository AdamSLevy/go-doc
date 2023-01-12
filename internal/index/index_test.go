package index

import (
	"bytes"
	"encoding/json"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// func init() { dlog.Enable() }

type searchTest struct {
	Name     string
	Opts     []SearchOption
	Searches []string
	Expected []string
}

type packagesTest struct {
	Name        string
	Modules     []Module
	SearchTests []searchTest
}

var tests = []packagesTest{{
	Name:    "empty",
	Modules: nil,
	SearchTests: []searchTest{{
		Opts:     []SearchOption{SearchExact()},
		Searches: []string{"foo"},
		Expected: nil,
	}, {
		Searches: []string{"foo"},
		Expected: nil,
	}},
}, {
	Name: "one module",
	Modules: []Module{newModule("foo.com/bar", "v1.0.0",
		"", "foo", "fib", "baz", "bif", "internal/foo", "internal/fib", "internal/baz", "internal/bif")},
	SearchTests: []searchTest{{
		Opts:     []SearchOption{SearchExact()},
		Searches: []string{"foo.com/bar", "bar"},
		Expected: []string{"foo.com/bar"},
	}, {
		Opts:     []SearchOption{SearchExact()},
		Searches: []string{"foo"},
		Expected: []string{"foo.com/bar/foo", "foo.com/bar/internal/foo"},
	}, {
		Searches: []string{"fo"},
		Expected: []string{"foo.com/bar/foo", "foo.com/bar/internal/foo"},
	}, {
		Searches: []string{"b"},
		Expected: []string{"foo.com/bar", "foo.com/bar/baz", "foo.com/bar/bif", "foo.com/bar/internal/baz", "foo.com/bar/internal/bif"},
	}},
}}

func TestPackagesSearch(t *testing.T) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			pkgIndex := New()
			outdated := pkgIndex.Sync(test.Modules)
			require.Equal(t, test.Modules, outdated)
			pkgIndex.Update(test.Modules)

			buf := bytes.NewBuffer(nil)
			enc := json.NewEncoder(buf)
			enc.SetIndent("", "  ")
			require.NoError(t, enc.Encode(pkgIndex))
			t.Log("pkgIndex:")
			t.Log(buf.String())

			for _, searchTest := range test.SearchTests {
				t.Run(searchTest.Name, func(t *testing.T) {
					for _, search := range searchTest.Searches {
						t.Run(search, func(t *testing.T) {
							pkgs := pkgIndex.Search(search, searchTest.Opts...)
							require.Equal(t, searchTest.Expected, ToImportPaths(pkgs))
						})
					}
				})
			}
		})
	}
}

func newModule(importPath, version string, packages ...string) Module {
	mod := NewModule(importPath, strings.Join([]string{importPath, version}, "@"))
	for _, pkg := range packages {
		pkgPath := path.Join(importPath, pkg)
		mod.AddPackage(pkgPath, pkgPath)
	}
	return mod
}
