package index

import (
	"bytes"
	"encoding/json"
	"path"
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
			var pkgIndex packageIndex
			for _, m := range test.Modules {
				pkgIndex.Update(m)
			}

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
							require.Equal(t, searchTest.Expected, pkgs)
						})
					}
				})
			}
		})
	}
}

func newModule(importPath, version string, packages ...string) Module {
	for i := range packages {
		packages[i] = path.Join(importPath, packages[i])
	}
	return Module{
		ImportPath: importPath,
		Version:    version,
		Packages:   packages,
	}
}
