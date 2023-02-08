package index

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	_ "modernc.org/sqlite"

	"aslevy.com/go-doc/internal/godoc"
)

type Mode = string

const (
	ModeOff       Mode = "off"
	ModeAutoSync       = "auto"
	ModeForceSync      = "force"
	ModeSkipSync       = "skip"
)

func modes() string {
	return strings.Join([]Mode{ModeOff, ModeAutoSync, ModeForceSync, ModeSkipSync}, ", ")
}

func ParseMode(s string) (Mode, error) {
	switch s {
	case ModeOff, ModeAutoSync, ModeForceSync, ModeSkipSync:
		return s, nil
	}
	return ModeAutoSync, fmt.Errorf("invalid index mode: %q", s)
}

type Option func(*options)
type options struct {
	mode               Mode
	resyncInterval     time.Duration
	disableProgressBar bool
}

func newOptions(opts ...Option) options {
	o := defaultOptions()
	WithOptions(opts...)(&o)
	return o
}
func defaultOptions() options {
	return options{
		mode:           ModeAutoSync,
		resyncInterval: DefaultResyncInterval,
	}
}

func WithOptions(opts ...Option) Option {
	return func(o *options) {
		for _, opt := range opts {
			opt(o)
		}
	}
}

func WithAuto() Option      { return WithMode(ModeAutoSync) }
func WithOff() Option       { return WithMode(ModeOff) }
func WithForceSync() Option { return WithMode(ModeForceSync) }
func WithSkipSync() Option  { return WithMode(ModeSkipSync) }
func WithMode(mode Mode) Option {
	return func(o *options) {
		o.mode = mode
	}
}

func WithResyncInterval(interval time.Duration) Option {
	return func(o *options) {
		o.resyncInterval = interval
	}
}

func WithNoProgressBar() Option {
	return func(o *options) {
		o.disableProgressBar = true
	}
}

type Index struct {
	options

	db *sql.DB
	tx *sqlTx

	sync
	cancel context.CancelFunc
	g      *errgroup.Group
}

type sqlTx struct {
	*sql.Tx
	stmts map[string]*sql.Stmt
}

func newSqlTx(tx *sql.Tx) *sqlTx {
	return &sqlTx{
		Tx:    tx,
		stmts: make(map[string]*sql.Stmt),
	}
}

func (tx *sqlTx) Prepare(query string) (*sql.Stmt, error) {
	return tx.PrepareContext(context.Background(), query)
}
func (tx *sqlTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	stmt, ok := tx.stmts[query]
	if ok {
		return stmt, nil
	}
	stmt, err := tx.Tx.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	tx.stmts[query] = stmt
	return stmt, nil
}

func Load(ctx context.Context, dbPath string, codeRoots []godoc.PackageDir, opts ...Option) (*Index, error) {
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil, nil
	}

	dlog.Printf("loading %q", dbPath)
	dlog.Printf("options: %+v", o)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index database: %w", err)
	}

	idx := Index{
		options: o,
		db:      db,
	}

	if err := idx.checkSetApplicationID(ctx); err != nil {
		return nil, err
	}

	ctx, idx.cancel = context.WithCancel(ctx)
	idx.g, ctx = errgroup.WithContext(ctx)
	idx.g.Go(func() error {
		defer idx.cancel()
		return idx.initSync(ctx, codeRoots)
	})

	return &idx, nil
}

func (idx *Index) Close() error {
	idx.cancel()
	if err := idx.waitSync(); err != nil {
		dlog.Printf("failed to sync: %v", err)
	}
	return idx.db.Close()
}
func (idx *Index) waitSync() error { return idx.g.Wait() }

func (idx *Index) initSync(ctx context.Context, codeRoots []godoc.PackageDir) error {
	if err := idx.enableForeignKeys(ctx); err != nil {
		return err
	}

	if err := idx.applySchema(ctx); err != nil {
		return err
	}

	return idx.syncCodeRoots(ctx, codeRoots)
}
