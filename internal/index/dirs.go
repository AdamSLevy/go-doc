package index

import (
	"context"

	"aslevy.com/go-doc/internal/godoc"
	"golang.org/x/sync/errgroup"
)

type Dirs struct {
	idx *Index

	searchPath    string
	searchPartial bool
	g             *errgroup.Group
	cancel        context.CancelFunc

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
	if d.searchPath == path && d.searchPartial == partial {
		return nil
	}

	if err := d.idx.waitSync(); err != nil {
		return err
	}

	d.Reset()
	if d.cancel != nil {
		d.cancel()
		d.g.Wait()
	}
	d.results = d.results[:0]

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	d.cancel = cancel

	rows, err := d.idx.searchRows(ctx, path, partial)
	if err != nil {
		return err
	}

	d.searchPath = path
	d.searchPartial = partial
	d.next = make(chan godoc.PackageDir)

	d.g, ctx = errgroup.WithContext(ctx)
	d.g.Go(func() error {
		defer cancel()
		defer close(d.next)
		return scanPackageDirs(rows, func(pkg godoc.PackageDir) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case d.next <- pkg:
			}
			return nil
		})
	})

	return nil
}
