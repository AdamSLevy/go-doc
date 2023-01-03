package completion

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartialSegments(t *testing.T) {
	tests := []struct {
		partial  string
		segments []string
	}{{
		partial:  "...",
		segments: []string{""},
	}, {
		partial:  "/",
		segments: []string{""},
	}, {
		partial:  "/...",
		segments: []string{""},
	}, {
		partial:  "/.../...",
		segments: []string{""},
	}, {
		partial:  "/.../.../",
		segments: []string{""},
	}, {
		partial:  "a/b/c",
		segments: []string{"a", "b", "c"},
	}, {
		partial:  ".../a/b/c",
		segments: []string{"a", "b", "c"},
	}, {
		partial:  "/a/b/c",
		segments: []string{"a", "b", "c"},
	}, {
		partial:  "a/b/.../c",
		segments: []string{"a", "b", "...", "c"},
	}, {
		partial:  "a/b/.../.../c",
		segments: []string{"a", "b", "...", "c"},
	}, {
		partial:  "a/b/...//c",
		segments: []string{"a", "b", "...", "", "c"},
	}, {
		partial:  "a/b/.../c/...",
		segments: []string{"a", "b", "...", "c", ""},
	}}
	for _, test := range tests {
		t.Run(test.partial, func(t *testing.T) {
			segments := partialSegments(test.partial)
			require.Equal(t, test.segments, segments, test.partial)
		})
	}
}

func TestMatchPackage(t *testing.T) {
	type packageMatch struct {
		pkgPath string
		match   string
	}
	type matchPackageTest struct {
		partial  string
		packages []packageMatch
	}

	tests := []matchPackageTest{{
		partial: "c",
		packages: []packageMatch{{
			pkgPath: "at/bat/cat",
			match:   "cat",
		}},
	}, {
		partial: "b/c",
		packages: []packageMatch{{
			pkgPath: "at/bat/cat",
			match:   "bat/cat",
		}},
	}, {
		partial: "",
		packages: []packageMatch{{
			pkgPath: "at/bat/cat",
			match:   "cat",
		}},
	}, {
		partial: "j",
		packages: []packageMatch{{
			pkgPath: "encoding/json",
			match:   "json",
		}, {
			pkgPath: "image/jpeg",
			match:   "jpeg",
		}, {
			pkgPath: "net/rpc/jsonrpc",
			match:   "jsonrpc",
		}, {
			pkgPath: "at/bat/cat",
		}},
	}, {
		partial: "e/j",
		packages: []packageMatch{{
			pkgPath: "encoding/json",
			match:   "encoding/json",
		}, {
			pkgPath: "image/jpeg",
		}, {
			pkgPath: "net/rpc/jsonrpc",
		}, {
			pkgPath: "at/bat/cat",
		}},
	}, {
		partial: "e/.../j",
		packages: []packageMatch{{
			pkgPath: "encoding/json",
			match:   "encoding/json",
		}, {
			pkgPath: "encoding/a/b/c/json",
			match:   "encoding/a/b/c/json",
		}, {
			pkgPath: "image/jpeg",
		}, {
			pkgPath: "net/rpc/jsonrpc",
		}, {
			pkgPath: "at/bat/cat",
		}},
	}, {
		partial: "a/b/.../f/g/.../j",
		packages: []packageMatch{{
			pkgPath: "a/b/c/d/e/f/g/h/i/j",
			match:   "a/b/c/d/e/f/g/h/i/j",
		}, {
			pkgPath: "a/b/c/d/e/f/g/h/i/j/j/j",
			match:   "a/b/c/d/e/f/g/h/i/j/j/j",
		}, {
			pkgPath: "z/a/b/c/d/e/f/g/h/i/j/k/l",
			match:   "a/b/c/d/e/f/g/h/i/j/k/l",
		}, {
			pkgPath: "image/jpeg",
		}, {
			pkgPath: "net/rpc/jsonrpc",
		}, {
			pkgPath: "at/bat/cat",
		}},
	}, {
		partial: "g//.../sp",
		packages: []packageMatch{{
			pkgPath: "github.com/davecgh/go-spew/spew",
			match:   "github.com/davecgh/go-spew/spew",
		}},
	}, {
		partial: "g/...//.../sp",
		packages: []packageMatch{{
			pkgPath: "github.com/davecgh/go-spew/spew",
			match:   "github.com/davecgh/go-spew/spew",
		}},
	}}
	for _, test := range tests {
		t.Run(test.partial, func(t *testing.T) {
			partials := partialSegments(test.partial)
			for _, pkg := range test.packages {
				name := "should match/"
				if pkg.match == "" {
					name = "should not match/"
				}
				t.Run(name+pkg.pkgPath, func(t *testing.T) {
					segments := strings.Split(pkg.pkgPath, "/")
					match := matchPackage(segments, countGlobs(partials...), partials...)
					require.Equal(t, pkg.match, match)
				})
			}
		})
	}
}
