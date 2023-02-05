package main

import (
	"aslevy.com/go-doc/internal/godoc"
)

var xdirs godoc.Dirs = dirs.PackageDirs()

func (dirs *Dirs) PackageDirs() *PackageDirs { return (*PackageDirs)(dirs) }

type PackageDirs Dirs

func (d *PackageDirs) dirs() *Dirs { return (*Dirs)(d) }

func (d *PackageDirs) Next() (godoc.PackageDir, bool) {
	dir, ok := d.dirs().Next()
	return godoc.NewPackageDir(dir.importPath, dir.dir), ok
}

func (d *PackageDirs) Reset() { d.dirs().Reset() }

func (d *PackageDirs) Filter(string, bool) error { return godoc.ErrFilterNotSupported }

func (dirs *Dirs) registerPackage(importPath, dir string) { dirs.scan <- Dir{importPath, dir, true} }
