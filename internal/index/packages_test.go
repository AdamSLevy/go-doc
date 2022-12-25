package index

import (
	"sort"
	"testing"
)

func TestPackages(t *testing.T) {
	var pkgs Packages

	sort.Strings(paths)
	for _, path := range paths {
		pkgs.Insert(path, "")
	}

	t.Log(pkgs)

	partial := "a"
	t.Logf("partial: %s", partial)
	t.Log("matches:")
	matches := pkgs.Matches(partial)
	for _, match := range matches {
		t.Log(match)
	}
}

var paths = []string{
	"aa",
	"ab",
	"ba",
	"bb",

	"aa/aa",
	"aa/ab",
	"aa/ba",
	"aa/bb",

	"ab/aa",
	"ab/ab",
	"ab/ba",
	"ab/bb",

	"ba/aa",
	"ba/ab",
	"ba/ba",
	"ba/bb",

	"bb/aa",
	"bb/ab",
	"bb/ba",
	"bb/bb",

	"aa/aa/aa",
	"aa/aa/ab",
	"aa/aa/ba",
	"aa/aa/bb",

	"aa/ab/aa",
	"aa/ab/ab",
	"aa/ab/ba",
	"aa/ab/bb",

	"aa/ba/aa",
	"aa/ba/ab",
	"aa/ba/ba",
	"aa/ba/bb",

	"aa/bb/aa",
	"aa/bb/ab",
	"aa/bb/ba",
	"aa/bb/bb",

	"ab/aa/aa",
	"ab/aa/ab",
	"ab/aa/ba",
	"ab/aa/bb",

	"ab/ab/aa",
	"ab/ab/ab",
	"ab/ab/ba",
	"ab/ab/bb",

	"ab/ba/aa",
	"ab/ba/ab",
	"ab/ba/ba",
	"ab/ba/bb",

	"ab/bb/aa",
	"ab/bb/ab",
	"ab/bb/ba",
	"ab/bb/bb",

	"ba/aa/aa",
	"ba/aa/ab",
	"ba/aa/ba",
	"ba/aa/bb",

	"ba/ab/aa",
	"ba/ab/ab",
	"ba/ab/ba",
	"ba/ab/bb",

	"ba/ba/aa",
	"ba/ba/ab",
	"ba/ba/ba",
	"ba/ba/bb",

	"ba/bb/aa",
	"ba/bb/ab",
	"ba/bb/ba",
	"ba/bb/bb",

	"bb/aa/aa",
	"bb/aa/ab",
	"bb/aa/ba",
	"bb/aa/bb",

	"bb/ab/aa",
	"bb/ab/ab",
	"bb/ab/ba",
	"bb/ab/bb",

	"bb/ba/aa",
	"bb/ba/ab",
	"bb/ba/ba",
	"bb/ba/bb",

	"bb/bb/aa",
	"bb/bb/ab",
	"bb/bb/ba",
	"bb/bb/bb",
}
