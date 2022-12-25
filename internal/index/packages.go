package index

import (
	"fmt"
	"sort"
	"strings"
)

// TODO:
// - docs
// - load/save to text or binary format
// - robust tests
// - comparison with default package resolution
// - globbing

type Packages struct {
	PackageDir
	IsPackage bool
	Children  []*Packages
}

type PackageDir struct {
	Path string
	Dir  string
}

func (pd PackageDir) String() string { return pd.Path }

func (ps *Packages) Insert(path, dir string) {
	if len(path) == len(ps.Path) {
		ps.IsPackage = true
		ps.Dir = dir
		return
	}

	slash := strings.LastIndex(path[:len(path)-len(ps.Path)-1], "/")
	childPath := path[slash+1:]

	// Search for an existing child path segment.
	for _, child := range ps.Children {
		if child.Path == childPath {
			child.Insert(path, dir)
			return
		}
	}

	// Create a new child.
	child := Packages{PackageDir: PackageDir{Path: childPath}}
	child.Insert(path, dir)

	ps.Children = append(ps.Children, &child)
}

func (ps Packages) String() string {
	var bldr strings.Builder
	ps.render(&bldr)
	return bldr.String()
}
func (ps *Packages) render(bldr *strings.Builder) {
	var pkg string
	if ps.IsPackage {
		pkg = "package"
	}
	fmt.Fprintln(bldr, ps.Path, pkg)
	for _, child := range ps.Children {
		child.render(bldr)
	}
}

func (ps *Packages) Matches(partial string) (matches []PackageDir) {
	return ps.matches(partial, true)
}
func (ps *Packages) matches(partial string, descend bool) (matches []PackageDir) {
	if partial == "" && ps.IsPackage {
		matches = append(matches, ps.PackageDir)
	}

	slash := strings.LastIndex(partial, "/")
	rightPartial := partial[slash+1:]
	remainingPartial := strings.TrimSuffix(partial[:slash+1], "/")

	var secondOrder []PackageDir
	// Search for an existing child partial segment.
	for _, child := range ps.Children {
		if strings.HasPrefix(child.Path, rightPartial) {
			matches = append(matches, child.matches(remainingPartial, false)...)
			continue
		}
		if !descend {
			continue
		}
		secondOrder = append(secondOrder, child.Matches(partial)...)
	}

	byLen := func(i, j int) bool { return len(matches[i].Path) < len(matches[j].Path) }
	matches = append(matches, secondOrder...)
	sort.SliceStable(matches, byLen)
	return matches
}
