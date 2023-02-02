package index

import (
	"context"

	"aslevy.com/go-doc/internal/godoc"
	"golang.org/x/sync/errgroup"
)

type Dirs struct {
	idx *Index

	searchPath string
	path       bool
	g          *errgroup.Group
	cancel     context.CancelFunc

	next    chan godoc.PackageDir
	results []godoc.PackageDir
	offset  int
}

var _ godoc.Dirs = (*Dirs)(nil)

func NewDirs(pkgIdx *Index) godoc.Dirs {
	return &Dirs{idx: pkgIdx}
}

func (d *Dirs) Reset() { d.offset = 0 }
func (d *Dirs) Next() (pkg godoc.PackageDir, ok bool) {
	if d.offset < len(d.results) {
		pkg := d.results[d.offset]
		d.offset++
		return pkg, true
	}

	pkg, ok = <-d.next
	if ok {
		d.results = append(d.results, pkg)
		d.offset++
	}
	return pkg, ok
}
func (d *Dirs) Filter(path string, partial bool) error {
	if d.searchPath == path && d.path == partial {
		return nil
	}
	if err := d.idx.waitSync(); err != nil {
		return err
	}

	d.Reset()

	ctx := context.Background()
	rows, err := d.idx.searchRows(ctx, path, partial)
	if err != nil {
		return err
	}
	d.searchPath = path
	d.path = partial
	d.next = make(chan godoc.PackageDir)

	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.g, ctx = errgroup.WithContext(ctx)
	d.g.Go(func() error {
		defer cancel()
		defer rows.Close()
		defer close(d.next)
		for rows.Next() && ctx.Err() == nil {
			pkg, err := scanPackageDir(rows)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case d.next <- pkg:
			}
		}

		return rows.Err()
	})
	return nil
}
