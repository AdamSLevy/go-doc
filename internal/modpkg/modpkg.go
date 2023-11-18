package modpkg

import (
	"context"
	"errors"
	"fmt"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/modpkg/db"
	modpkgdb "aslevy.com/go-doc/internal/modpkg/db"
	"golang.org/x/sync/errgroup"
)

type ModPkg struct {
	db *db.DB
	g  *errgroup.Group
}

func New(ctx context.Context, GOROOT, GOMODCACHE, GOMOD string, coderoots []godoc.PackageDir) (*ModPkg, error) {
	db, err := modpkgdb.OpenDB(ctx, GOROOT, GOMODCACHE, GOMOD)
	if err != nil {
		return nil, err
	}

	modPkg := ModPkg{db: db}
	modPkg.g, ctx = errgroup.WithContext(ctx)
	modPkg.g.Go(func() error { return modPkg.sync(ctx, coderoots) })

	return &modPkg, nil
}

func (modPkg *ModPkg) sync(ctx context.Context, coderoots []godoc.PackageDir) (rerr error) {
	sync, err := modPkg.db.StartSyncIfNeeded(ctx)
	if err != nil {
		return err
	}
	if sync == nil {
		return nil
	}
	defer func() {
		if err := sync.Finish(ctx); err != nil {
			rerr = errors.Join(rerr, fmt.Errorf("failed to finish sync: %w", err))
		}
	}()

	if sync.Meta.Vendor {
		return modPkg.syncVendored(ctx, sync, coderoots)
	}
	return modPkg.syncGoModCache(ctx, sync, coderoots)
}

func (modPkg *ModPkg) Close() error {
	rerr := modPkg.g.Wait()
	if err := modPkg.db.Close(); err != nil {
		rerr = errors.Join(rerr, fmt.Errorf("failed to close module/package database: %w", err))
	}
	return rerr
}
