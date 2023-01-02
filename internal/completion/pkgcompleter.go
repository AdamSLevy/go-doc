package completion

import (
	"go/build"
	"io/fs"
	"path/filepath"
	"strings"

	"aslevy.com/go-doc/internal/dlog"
)

func (c Completer) completePackages(partial string) (matched bool) {
	// Paths which start with a dot or use the backslash cannot be package
	// import paths. Note that paths starting with a slash could be
	// a partial import path, so we don't exclude them.
	invalidImportPath := strings.Contains(partial, `\`) || // must be a file path on Windows
		(strings.HasPrefix(partial, ".") && !strings.HasPrefix(partial, ".../")) // must be a relative path
	if !invalidImportPath && c.completePackageImportPaths(partial) {
		return true
	}

	// complete file paths since we didn't match any packages.
	return c.completePackageFilePaths(partial)
}

func (c Completer) completePackageImportPaths(partial string) (matched bool) {
	dlog.Printf("completing package import paths matching %q", partial)

	partials := partialSegments(partial)
	dlog.Printf("partials: %v", partials)
	shortPaths := make(ShortImportPaths)

	// list all possible packages
	c.dirs.Reset()
	for {
		dir, ok := c.dirs.Next()
		if !ok {
			break
		}

		importPathSegments := strings.Split(dir.ImportPath, "/")
		shortPath := dir.ImportPath
		if ShortPath {
			shortPath = shortPaths.ShortestUniqueImportPath(importPathSegments...)
		}

		match := matchPackage(importPathSegments, countGlobs(partials...), partials...)
		if match == "" {
			// Not a match.
			continue
		}

		desc, ok := describePackage(dir.Dir)
		if !ok {
			// Not a real package, so we can remove its unique
			// short path.
			delete(shortPaths, shortPath)
			continue
		}

		matched = true

		// Use the shortest unique path unless it is shorter than what
		// the user already specified.
		if len(shortPath) > len(match) {
			match = shortPath
		}

		c.suggest(NewMatch(
			match,
			WithDisplay(dir.ImportPath),
			WithDescription(desc),
			WithTag(TagPackages),
		))
	}
	return
}

func partialSegments(partial string) []string {
	segments := strings.Split(partial, "/")
	partials := make([]string, 0, len(segments))
	for _, segment := range segments {
		switch segment {
		case "":
			// partials cannot start with an empty segment
			if len(partials) == 0 {
				continue
			}
		case "...":
			// partials cannot start with "..." nor contain
			// consecutive "..."
			if len(partials) == 0 || last(partials) == "..." {
				continue
			}
		}
		partials = append(partials, segment)
	}
	if len(partials) == 0 {
		return []string{""}
	}
	if len(partials) > 0 && last(partials) == "..." {
		partials[lastIdx(partials)] = ""
	}
	return partials
}

func (c Completer) completePackageFilePaths(partial string) (matched bool) {
	dlog.Printf("completing local paths to packages matching %q", partial)

	partial = filepath.FromSlash(partial)
	partial = filepath.Clean(partial)
	sep := string(filepath.Separator)
	partials := strings.Split(partial, sep)
	var isAbs, isDotDot bool
	for i, partial := range partials {
		if i == 0 && partial == "" {
			isAbs = true
			continue
		}
		if partial == "." {
			continue
		}
		if partial == ".." {
			isDotDot = true
			continue
		}
		partials[i] += "*"
	}
	pattern := strings.Join(partials, sep)
	dlog.Printf("globbing for paths matching %q", pattern)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	const maxDepth = 5
	walked := make(map[string]struct{}, len(matches))
	// Find all pkgDirs within our matches which contain at least one go
	// file.
	var pkgDirs []string
	for _, match := range matches {
		dlog.Printf("walking %q", match)
		var lastDir string
		numSlash := strings.Count(match, "/")
		filepath.WalkDir(match, func(path string, d fs.DirEntry, _ error) error {
			name := d.Name()
			if !d.IsDir() {
				if dir := filepath.Dir(path); lastDir != dir && strings.HasSuffix(name, ".go") {
					pkgDirs = append(pkgDirs, dir)
					lastDir = dir
				}
				return nil
			}
			if name == ".." || name == "." {
				return nil
			}
			if _, seen := walked[path]; seen {
				return fs.SkipDir
			}
			walked[path] = struct{}{}
			if name[0] == '.' ||
				name == "vendor" ||
				strings.Count(path, "/")-numSlash > maxDepth {
				return fs.SkipDir
			}
			return nil
		})
	}
	suggested := make(map[string]struct{}, len(pkgDirs))
	for _, pkgDir := range pkgDirs {
		if _, ok := suggested[pkgDir]; ok {
			continue
		}
		suggested[pkgDir] = struct{}{}

		if pkgDir == "." {
			continue
		}

		desc, ok := describePackage(pkgDir)
		if !ok {
			continue
		}

		matched = true

		if !isDotDot && !isAbs {
			pkgDir = "." + sep + pkgDir
		}

		c.suggest(NewMatch(pkgDir, WithDescription(desc), WithTag(TagPackages)))
	}
	return
}

func describePackage(packageDir string) (string, bool) {
	pkg, err := build.ImportDir(packageDir, build.ImportComment)
	if err != nil {
		return "", false
	}
	docs := firstSentence(pkg.Doc)
	docs = trimPackagePrefix(docs, pkg.Name)
	if docs == "" {
		docs = "Package " + pkg.Name
	}
	return docs, true
}
func firstSentence(docs string) string {
	// Get the first paragraph.
	docs, _, _ = strings.Cut(docs, "\n\n")
	// Join all lines.
	docs = strings.ReplaceAll(docs, "\n", " ")
	// Get the first sentence.
	docs, _, found := strings.Cut(docs, ". ")
	if found {
		return docs
	}
	// The first paragraph may have been a single sentence with a newline
	// instead of a space after the period. Remove any trailing period for
	// consistency.
	docs = strings.TrimSuffix(docs, ".")
	return docs
}
func trimPackagePrefix(docs, pkgName string) string {
	if d := strings.TrimPrefix(docs, "Package "); d != docs {
		docs = d
	} else {
		docs = strings.TrimPrefix(docs, "package ")
	}
	docs = strings.TrimPrefix(docs, pkgName)
	docs = strings.TrimPrefix(docs, " ")
	return docs
}

type ShortImportPaths map[string]struct{}

func (s ShortImportPaths) ShortestUniqueImportPath(pathSegments ...string) string {
	// TODO: figure out a way to avoid non-package directories consuming
	// unique short paths without loading the docs for every package path.
	var short string
	for i := range pathSegments {
		short = strings.Join(pathSegments[lastIdx(pathSegments)-i:], "/")
		if _, taken := s[short]; !taken {
			break
		}
	}
	s[short] = struct{}{}
	return short
}

func matchPackage(segments []string, numGlobs int, partials ...string) string {
	excludeInternal := false
	allSegments := segments
	allPartials := partials
	var firstMatchIdx int
	var foundFirstMatch bool
	for len(partials) > 0 && len(segments) >= len(partials)-numGlobs {
		var partial string
		_, partial, partials = popEnd(partials)

		if partial == "..." {
			// glob
			match := matchPackage(segments, numGlobs-1, partials...)
			if match == "" {
				return ""
			}
			return match + "/" + strings.Join(allSegments[len(segments):], "/")
		}

		var segmentIdx int
		var segment string
		segmentIdx, segment, segments = popEnd(segments)
		if strings.HasPrefix(segment, partial) {
			if !foundFirstMatch {
				firstMatchIdx = segmentIdx
				foundFirstMatch = true
			}
			continue
		}
		// The segment is not a match for this partial.

		if excludeInternal && segment == "internal" {
			// Failing to match the "internal" segment
			// excludes the package.
			return ""
		}

		partials = append(partials, partial)

		if !foundFirstMatch {
			continue
		}

		// We failed to match.
		break
	}
	if !foundFirstMatch {
		return ""
	}
	if len(partials) > 0 {
		match := matchPackage(allSegments[:firstMatchIdx], numGlobs, allPartials...)
		if match == "" {
			return ""
		}
		return match + "/" + strings.Join(allSegments[firstMatchIdx:], "/")
	}

	if excludeInternal {
		for _, segment := range allSegments[:len(segments)] {
			if segment == "internal" {
				// There is an unmatched preceding internal segment.
				return ""
			}
		}
	}

	// Return the tail of the import path starting from the segments
	// matched by the partials. This allows us to preserve all path
	// segments provided by the user, when longer than the shortest unique
	// import path.
	return strings.Join(allSegments[len(segments):], "/")
}

func countGlobs(partials ...string) int {
	var numGlobs int
	for _, partial := range partials {
		if partial == "..." {
			numGlobs++
		}
	}
	return numGlobs
}

func lastIdx[T any](ts []T) int {
	return len(ts) - 1
}
func last[T any](ts []T) T {
	return ts[lastIdx(ts)]
}
func lastOffset[T any](ts []T, i int) T {
	return ts[lastIdx(ts)-i]
}
func popEnd[T any](ts []T) (int, T, []T) {
	idx := lastIdx(ts)
	return idx, ts[idx], ts[:idx]
}
