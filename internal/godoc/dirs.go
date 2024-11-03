// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package godoc

import (
	"errors"
	"path/filepath"
	"strings"
)

// A PackageDir describes a directory holding code by specifying the expected
// import path and the file system directory.
type PackageDir struct {
	ImportPath string // import path for that dir
	Dir        string // file system directory
	Version    string // module version (if applicable)
}

type PackageDirOption func(*PackageDir)

func WithVersion(version string) PackageDirOption {
	return func(pkg *PackageDir) {
		pkg.Version = version
	}
}

func NewPackageDir(importPath, dir string, opts ...PackageDirOption) PackageDir {
	pkgDir := PackageDir{
		ImportPath: importPath,
		Dir:        dir,
	}
	for _, opt := range opts {
		opt(&pkgDir)
	}
	if pkgDir.Version == "" {
		pkgDir.Version = parseVersionFromDir(dir)
	}
	return pkgDir
}
func parseVersionFromDir(dir string) string {
	base := filepath.Base(dir)
	_, version, _ := strings.Cut(base, "@")
	return version
}

// Dirs exposes the functionality of the cmd/go-doc.Dirs type that is needed by
// the completion package.
type Dirs interface {
	// Next returns the next PackageDir in the list of packages.
	Next() (PackageDir, bool)

	// Reset resets Next to the beginning of the list of packages.
	// A subsequent call to Next will return the first PackageDir.
	Reset()

	FilterExact(path string) error
	FilterPartial(path string) error
}

var ErrFilterNotSupported = errors.New("filter not supported")
