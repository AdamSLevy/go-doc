// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godoc

// A Dir describes a directory holding code by specifying
// the expected import path and the file system directory.
type Dir struct {
	ImportPath string // import path for that dir
	Dir        string // file system directory
	InModule   bool
}

// PackageDirs exposes the functionality of the cmd/go-doc.Dirs type that is
// needed by the completion package.
type PackageDirs interface {
	// findNextPackage returns the next full file name path that matches
	// the (perhaps partial) package path pkg. The boolean reports if any
	// match was found.
	//
	// A pkg is a partial of a package's import path if "/"+pkg is a suffix
	// of the import path. For example, "json" is a partial of
	// "encoding/json", but not of path/to/endswithjson.
	//
	// Matching is case sensitive.
	FindNextPackage(pkg string) (string, bool)

	// Next returns the next Dir in the list of packages.
	Next() (Dir, bool)

	// Reset resets Next to the beginning of the list of packages.
	// A subsequent call to Next will return the first Dir.
	Reset()
}
