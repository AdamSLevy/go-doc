package index

import (
	"encoding/json"

	"golang.org/x/exp/slices"

	islices "aslevy.com/go-doc/internal/slices"
)

// rightPartial groups packages which share the same right-most path components
// of their import path.
type rightPartial struct {
	// CommonParts are the right-most segments of the import paths common
	// to all Packages.
	CommonParts []string
	Packages    packageList
}

func newRightPartial(parts []string, pkgs ..._Package) rightPartial {
	return rightPartial{CommonParts: parts, Packages: pkgs}
}

func (part *rightPartial) updatePackages(add bool, pkgs ..._Package) {
	part.Packages.Update(add, pkgs...)
}
func (p rightPartial) shouldOmit() bool { return len(p.Packages) == 0 }

// rightPartialList is a list of rightPartials which all share the same number
// of CommonParts, sorted by CommonParts.
//
// When marshaled to JSON, rightPartialList omits any rightPartial with no
// Packages.
type rightPartialList []rightPartial

func (p *rightPartialList) updatePartial(add bool, parts []string, pkgs ..._Package) {
	debug.Printf("partials.update(%q, %q)", parts, pkgs)
	newPart := newRightPartial(parts, pkgs...)
	*p, _, _ = islices.UpdateSorted(*p, newPart, comparePartials, islices.WithReplaceFunc[rightPartial](func(existing, _ *rightPartial) {
		existing.updatePackages(add, pkgs...)
	}))
}
func comparePartials(a, b rightPartial) int {
	return slices.CompareFunc(a.CommonParts, b.CommonParts, stringsCompare)
}

// stringsCompare is like strings.Compare.
//
// But apparently you're not supposed to use strings.Compare according to its
// docs.
func stringsCompare(a, b string) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

// MarshalJSON omits any rightPartial with no Packages.
func (pl rightPartialList) MarshalJSON() ([]byte, error) { return omitEmptyElementsMarshalJSON(pl) }

// rightPartialIndex is a list of rightPartialLists, in ascending
// order of the number of slashes in the right partials of each list.
//
// For example, the right partial "b/c" would be indexed in the second list,
// index 1.
type rightPartialIndex []rightPartialList

func (bns rightPartialIndex) MarshalJSON() ([]byte, error) {
	if len(bns) > 0 && len(bns[len(bns)-1]) == 0 {
		bns = bns[:len(bns)-1]
	}
	return json.Marshal([]rightPartialList(bns))
}

func (bns *rightPartialIndex) Insert(modParts []string, pkg _Package) {
	bns.Update(true, modParts, pkg)
}
func (bns *rightPartialIndex) Remove(modParts []string, pkg _Package) {
	bns.Update(false, modParts, pkg)
}
func (bns *rightPartialIndex) Update(add bool, modParts []string, pkg _Package) {
	debug.Printf("Packages.update(mod:%q, %q, %v)", pkg.ModulePath(), pkg.ImportPath(), add)
	parts := append(modParts, pkg.ImportPathParts[1:]...)
	if len(*bns) < len(parts) {
		*bns = append(*bns, make([]rightPartialList, len(parts)-len(*bns))...)
	}
	for i := range parts {
		bns.update(add, parts[i:], pkg)
	}
}
func (bns rightPartialIndex) update(add bool, parts []string, pkg _Package) {
	bns[len(parts)-1].updatePartial(add, parts, pkg)
}
