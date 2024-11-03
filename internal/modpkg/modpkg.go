package modpkg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
	"aslevy.com/go-doc/internal/progressbar"
	"golang.org/x/sync/errgroup"
)

type ModPkg struct {
	db *db.DB
	g  *errgroup.Group

	rows    *sql.Rows
	results []godoc.PackageDir
	offset  int

	search string
	exact  bool
}

func New(ctx context.Context, mainModDir string, coderoots []godoc.PackageDir) (*ModPkg, error) {
	db, err := db.Open(ctx, mainModDir)
	if err != nil {
		return nil, err
	}

	modPkg := ModPkg{db: db}
	modPkg.g, ctx = errgroup.WithContext(ctx)
	modPkg.g.Go(func() error { return modPkg.sync(ctx, coderoots) })

	return &modPkg, nil
}

func (modPkg *ModPkg) sync(ctx context.Context, coderoots []godoc.PackageDir) (rerr error) {

	sync, err := modPkg.db.Sync(ctx)
	if err != nil {
		return err
	}
	if sync == nil {
		// No need to sync.
		return nil
	}
	progressBar := progressbar.New(len(coderoots))
	defer func() {
		if err := sync.Finish(ctx); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to finish sync: %w", err))
		}
		if err := progressBar.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close progress bar: %w", err))
		}
	}()

	if sync.Current.Vendor {
		return modPkg.syncFromVendorDir(ctx, sync, coderoots[0])
	}
	return modPkg.syncFromGoModCache(ctx, progressBar, sync, coderoots)
}

func (modPkg *ModPkg) Close() error {
	var rerr error
	if modPkg.g != nil {
		if err := modPkg.g.Wait(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to wait for sync: %w", err))
		}
	} else if modPkg.rows != nil {
		if err := modPkg.rows.Close(); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to close previously open package search query: %w", err))
		}
	}
	if err := modPkg.db.Close(); err != nil {
		rerr = errors.Join(rerr, fmt.Errorf("failed to close module/package database: %w", err))
	}

	return rerr
}
