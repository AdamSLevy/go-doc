package index

import (
	"strings"
	"unicode"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/dlog"
)

type SearchOption func(*searchOptions)
type searchOptions struct {
	Exact bool
	Dirs  bool // Return the package directories instead of import paths.
}

func ReturnDirs() SearchOption  { return func(o *searchOptions) { o.Dirs = true } }
func SearchExact() SearchOption { return func(o *searchOptions) { o.Exact = true } }

func WithSearchOptions(opts ...SearchOption) SearchOption {
	return func(o *searchOptions) {
		for _, opt := range opts {
			opt(o)
		}
	}
}

func newSearchOptions(opts ...SearchOption) searchOptions {
	var o searchOptions
	WithSearchOptions(opts...)(&o)
	return o
}

func (p packageIndex) Search(path string, opts ...SearchOption) []string {
	parts := strings.Split(path, "/")
	numSlash := len(parts) - 1
	if numSlash >= len(p.Partials) {
		// We don't have any packages with this many slashes.
		return nil
	}

	var exactParts []string
	o := newSearchOptions(opts...)
	if o.Exact {
		exactParts = parts
		parts = nil
	}
	var pkgs packageList
	for _, partials := range p.Partials[numSlash:] {
		partials.searchPackages(&pkgs, exactParts, parts...)
		if o.Exact {
			break
		}
	}
	if o.Dirs {
		return pkgs.Dirs(p.Modules)
	}
	return pkgs.ImportPaths()
}

func (p rightPartialList) searchPackages(matches *packageList, exact []string, prefixes ...string) (pos int) {
	defer func() { dlog.Printf("searchPackages(%q, %q, %q): %v", matches, exact, prefixes, pos) }()

	var prefix string
	var prefixID int
	searchParts := exact
	if len(prefixes) > 0 {
		prefix = prefixes[0]
		searchParts = append(exact, prefix)
		prefixID = len(searchParts) - 1
	}

	var found bool
	pos, found = p.search(searchParts...)
	if prefix == "" {
		// We aren't searching for prefixes.
		if found {
			// We have an exact match.
			matches.Insert(p[pos].Packages...)
		}
		return
	}

	// pos is now the at the first partial that matches the exact parts and
	// the first prefix, if any such partial exists.
	for pos < len(p) {
		partial := p[pos]

		cmpExact := slices.CompareFunc(partial.CommonParts[:prefixID], exact, stringsCompare)
		if cmpExact > 0 {
			// We've gone past the exact match.
			return
		}

		hasPrefix := strings.HasPrefix(partial.CommonParts[prefixID], prefix) // Will always be true if prefix is empty.
		if !hasPrefix {
			// We've gone past the partials which match the first
			// prefix.
			return
		}

		// This partial matches the exact parts and the first prefix.

		exact := append(exact, partial.CommonParts[prefixID])
		prefixes := prefixes[1:]
		searchPos := p[pos:].searchPackages(matches, exact, prefixes...)
		pos += searchPos + 1

		// We need to search forward to the next prefix match.
		searchPos = p[pos:].searchPackages(nil, exact, string(unicode.MaxRune))
		pos += searchPos
	}
	return
}
func (p rightPartialList) search(parts ...string) (pos int, found bool) {
	return slices.BinarySearchFunc(p, rightPartial{CommonParts: parts}, comparePartials)
}
