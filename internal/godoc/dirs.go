// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godoc

// A PackageDir describes a directory holding code by specifying
// the expected import path and the file system directory.
type PackageDir struct {
	ImportPath string // import path for that dir
	Dir        string // file system directory
}

func NewPackageDir(importPath, dir string) PackageDir { return PackageDir{importPath, dir} }

// Dirs exposes the functionality of the cmd/go-doc.Dirs type that is
// needed by the completion package.
type Dirs interface {
	// Next returns the next PackageDir in the list of packages.
	Next() (PackageDir, bool)

	// Reset resets Next to the beginning of the list of packages.
	// A subsequent call to Next will return the first PackageDir.
	Reset()

	// Filter the list of packages to those that match the path either
	// exactly, or as prefixes of path segments. Subsequent calls to Next
	// will only return packages which match the filter, and Reset will
	// reset to the beginning of the filtered list.
	//
	// The returned value indicates whether the underlying implementation
	// supports this functionality. If false, the filter was not applied
	// and Next will iterate through all packages, leaving matching to the
	// caller.
	//
	// Multiple calls to Filter with the same path and exact values are
	// idempotent and will not affect the results of Next. If the path or
	// exact values differ from the previous call, the results of Next may
	// change, and Reset is called.
	Filter(path string, exact bool) bool
}
