package index

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index/schema"
)

type Index struct {
	options

	db *sql.DB

	schema.Metadata

	cancel context.CancelFunc
	g      *errgroup.Group
}

const DefaultRelativeDBPath = ".go-doc/packages.sqlite3"

func DBPath(mainModPath string) string {
	return filepath.Join(mainModPath, DefaultRelativeDBPath)
}

func Load(ctx context.Context, mainModPath string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil, nil
	}
	o.mainModPath = mainModPath
	if o.dbPath == "" {
		o.dbPath = DBPath(mainModPath)
	}

	dlog.Printf("loading database %s", o.dbPath)
	dlog.Printf("options: %+v", o)

	db, err := schema.OpenDB(ctx, o.dbPath)
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
