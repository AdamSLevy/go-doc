package index

import "aslevy.com/go-doc/internal/godoc"

type Dirs struct {
	pkgIdx *Packages

	path    string
	exact   bool
	results []godoc.PackageDir
	offset  int
}

var _ godoc.Dirs = (*Dirs)(nil)

func NewDirs(pkgIdx *Packages) godoc.Dirs {
	return &Dirs{pkgIdx: pkgIdx}
}

func (d *Dirs) Filter(path string, exact bool) bool {
	if d.path != path || d.exact != exact {
		*d = Dirs{
			pkgIdx: d.pkgIdx,

			path:    path,
			exact:   exact,
			results: d.pkgIdx.Search(path, exact),
			offset:  0,
		}
	}
	return true
}

func (d *Dirs) Reset() { d.offset = 0 }
func (d *Dirs) Next() (godoc.PackageDir, bool) {
	if d.offset >= len(d.results) {
		return godoc.PackageDir{}, false
	}
	d.offset++
	return d.results[d.offset-1], true
}
