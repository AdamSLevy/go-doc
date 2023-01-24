package index

import (
	"strings"
	"unicode"

	"aslevy.com/go-doc/internal/godoc"
	"golang.org/x/exp/slices"
)

func (p *Packages) Search(path string, exact bool) (results []godoc.PackageDir) {
	parts := strings.Split(path, "/")
	numSlash := len(parts) - 1
	if numSlash >= len(p.partials) {
		// We don't have any packages with this many slashes.
		return nil
	}

	var exactParts []string
	if exact {
		exactParts = parts
		parts = nil
	}
	var pkgs packageList
	for _, partials := range p.partials[numSlash:] {
		partials.searchPackages(&pkgs, exactParts, parts...)
		if exact {
			break
		}
		parts = append(parts, "") // Pad with empty string to search for prefixes.
	}
	return pkgs.PackageDirs(p.modules)
}

func (p rightPartialList) searchPackages(matches *packageList, exact []string, prefixes ...string) (pos int) {
	var numCommonParts int
	if len(p) > 0 {
		numCommonParts = len(p[0].CommonParts)
	}
	defer func() {
		debug.Printf("numCommonParts=%d searchPackages(%q, %q): %d", numCommonParts, exact, prefixes, pos)
	}()

	// The search parts are the exact parts and the first prefix, if any.
	searchParts := exact
	var firstPrefix string
	if len(prefixes) > 0 {
		firstPrefix = prefixes[0]
		searchParts = append(exact, firstPrefix)
	}

	var found bool
	pos, found = p.search(searchParts...)
	if len(prefixes) == 0 {
		// We aren't searching for prefixes.
		if found {
			// We have an exact match.
			matches.Insert(p[pos].Packages...)
		}
		return
	}

	// We are searching for prefixes.

	for pos < len(p) {
		maybe := p[pos]

		// Must match the exact parts.
		cmpExact := slices.CompareFunc(maybe.CommonParts[:len(exact)], exact, stringsCompare)
		if cmpExact > 0 {
			// We've gone past the exact match.
			return
		}

		// Must match the first prefix.
		hasPrefix := strings.HasPrefix(maybe.CommonParts[len(exact)], firstPrefix) // Will always be true if prefix is empty.
		if !hasPrefix {
			// We've gone past the partials which match the first
			// prefix.
			return
		}

		// We have a match for the exact parts and the first prefix.

		// Extend the exact parts with the part of this partial which
		// matches the first prefix.
		exact := append(exact, maybe.CommonParts[len(exact)])
		prefixes := prefixes[1:] // Drop the first prefix.
		// Recurse to search for the rest of the prefixes.
		pos += p[pos:].searchPackages(matches, exact, prefixes...)

		// We need to search forward to the next prefix match.
		pos += p[pos:].searchPackages(nil, exact, string(unicode.MaxRune))
	}
	return
}
func (p rightPartialList) search(parts ...string) (pos int, found bool) {
	return slices.BinarySearchFunc(p, rightPartial{CommonParts: parts}, comparePartials)
}
