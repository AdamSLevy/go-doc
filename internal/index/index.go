package index

import (
	"context"
	"fmt"
	"path/filepath"

	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkgdb"
)

type Index struct {
	options

	db *modpkgdb.DB

	modpkgdb.Metadata

	cancel context.CancelFunc
	g      *errgroup.Group
}

const DefaultRelativeDBPath = ".go-doc/packages.sqlite3"

func DBPath(mainModPath string) string {
	return filepath.Join(mainModPath, DefaultRelativeDBPath)
}

func Load(ctx context.Context, dbPath, goRootDir, goModCacheDir, mainModDir string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil, nil
	}
	o.goRootDir = goRootDir
	o.goModCacheDir = goModCacheDir
	o.mainModDir = mainModDir
	if o.dbPath == "" {
		o.dbPath = DBPath(mainModDir)
	}

	dlog.Printf("loading database %s", o.dbPath)
	dlog.Printf("options: %+v", o)

	db, err := modpkgdb.OpenDB(ctx, goRootDir, goModCacheDir, mainModDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	idx := Index{
		options: o,
		db:      db,
	}

	ctx, idx.cancel = context.WithCancel(ctx)
	idx.g, ctx = errgroup.WithContext(ctx)
	idx.g.Go(func() error {
		defer idx.cancel()
		return idx.syncCodeRoots(ctx, codeRoots)
	})

	return &idx, nil
}

func (idx *Index) waitSync() error { return idx.g.Wait() }

func (idx *Index) Close() error {
	idx.cancel()
	if err := idx.waitSync(); err != nil {
		dlog.Printf("failed to sync: %v", err)
	}
	return idx.db.Close()
}
