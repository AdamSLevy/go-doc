package index

import (
	"strings"
	"unicode"

	"golang.org/x/exp/slices"

	"aslevy.com/go-doc/internal/dlog"
)

type SearchOption func(*searchOptions)
type searchOptions struct{ Exact bool }

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

func (p *packageIndex) Search(path string, opts ...SearchOption) []string {
	parts := strings.Split(path, "/")
	numSlash := len(parts) - 1
	if numSlash >= len(p.ByNumSlash) {
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
	for _, partials := range p.ByNumSlash[numSlash:] {
		pkgs, _ = partials.searchPackages(pkgs, exactParts, parts...)
		if o.Exact {
			break
		}
	}
	return pkgs.ImportPaths()
}
func (p rightPartialList) search(parts ...string) (pos int, found bool) {
	return slices.BinarySearchFunc(p, rightPartial{Parts: parts}, comparePartials)
}

func (p rightPartialList) searchPackages(initial packageList, exact []string, prefixes ...string) (pkgs packageList, pos int) {
	defer func() {
		for _, pkg := range pkgs {
			initial = initial.Insert(pkg)
		}
		pkgs = initial
		dlog.Printf("searchPackages(%q, %q): %v", exact, prefixes, pkgs)
	}()

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
			pkgs = p[pos].Packages
		}
		return
	}

	// pos is now the at the first partial that matches the exact parts and
	// the first prefix, if any such partial exists.
	for pos < len(p) {
		partial := p[pos]

		cmpExact := slices.CompareFunc(partial.Parts[:prefixID], exact, stringsCompare)
		if cmpExact > 0 {
			// We've gone past the exact match.
			return
		}

		hasPrefix := strings.HasPrefix(partial.Parts[prefixID], prefix) // Will always be true if prefix is empty.
		if !hasPrefix {
			// We've gone past the partials which match the first
			// prefix.
			return
		}

		// This partial matches the exact parts and the first prefix.

		exact := append(exact, partial.Parts[prefixID])
		prefixes := prefixes[1:]
		var searchPos int
		pkgs, searchPos = p[pos:].searchPackages(pkgs, exact, prefixes...)
		pos += searchPos + 1

		// We need to search forward to the next prefix match.
		_, searchPos = p[pos:].searchPackages(nil, exact, string(unicode.MaxRune))
		pos += searchPos
	}
	return
}
