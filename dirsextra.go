package main

import "aslevy.com/go-doc/internal/godoc"

func (dirs *Dirs) PackageDirs() *PackageDirs { return (*PackageDirs)(dirs) }

type PackageDirs Dirs

func (d *PackageDirs) dirs() *Dirs { return (*Dirs)(d) }

func (d *PackageDirs) Next() (godoc.Dir, bool) {
	dir, ok := d.dirs().Next()
	return godoc.Dir{
		ImportPath: dir.importPath,
		Dir:        dir.dir,
	}, ok
}

func (d *PackageDirs) Reset() { d.dirs().Reset() }

func (dirs *Dirs) registerPackage(importPath, dir string) { dirs.scan <- Dir{importPath, dir, true} }
