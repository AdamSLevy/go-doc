package index

import (
	"encoding/json"
	"io"
	"os"
	"time"

	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
)

var Debug = dlog.Child("index")

const DefaultResyncInterval = 30 * time.Minute

var (
	disabled bool = func() bool { return os.Getenv("GODOC_DISABLE_INDEX") != "" }()

	Sync           Mode = ModeAutoSync
	ResyncInterval      = DefaultResyncInterval
)

type Packages struct {
	codeRoots []godoc.PackageDir
	modules   moduleList
	partials  rightPartialIndex

	createdAt time.Time
	updatedAt time.Time

	options
}

type options struct {
	mode               Mode
	resyncInterval     time.Duration
	disableProgressBar bool
}

type Option func(*options)

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

type Mode = string

const (
	ModeOff       Mode = "off"
	ModeAutoSync       = "auto"
	ModeForceSync      = "force"
	ModeSkipSync       = "skip"
)

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

func New(codeRoots []godoc.PackageDir, opts ...Option) *Packages {
	pkgIdx := newPackages(opts...)
	if pkgIdx == nil {
		return nil
	}
	pkgIdx.sync(codeRoots)
	return pkgIdx
}
func newPackages(opts ...Option) *Packages {
	if disabled {
		return nil
	}
	o := newOptions(opts...)
	if o.mode == ModeOff {
		return nil
	}
	return &Packages{
		createdAt: time.Now(),
		options:   o,
	}
}

func LoadSync(path string, required []godoc.PackageDir, opts ...Option) (pkgIdx *Packages, err error) {
	pkgIdx = newPackages(opts...)
	if pkgIdx == nil {
		return
	}

	defer func() {
		changed := pkgIdx.sync(required)
		if changed {
			err = pkgIdx.Save(path)
		}
	}()

	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	err = pkgIdx.decode(f)
	return
}
func (pkgIdx *Packages) decode(r io.Reader) error {
	var p packagesJSON
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return err
	}
	pkgIdx.fromPackagesJSON(p)
	return nil
}

func (pkgIdx *Packages) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	pkgIdx.updatedAt = time.Now()
	return pkgIdx.encode(f)
}
func (pkgIdx Packages) encode(w io.Writer) error {
	return json.NewEncoder(w).Encode(pkgIdx.toPackagesJSON())
}

type packagesJSON struct {
	CodeRoots []godoc.PackageDir
	Modules   moduleList
	Partials  rightPartialIndex
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (pkgIdx Packages) toPackagesJSON() packagesJSON {
	return packagesJSON{
		CodeRoots: pkgIdx.codeRoots,
		Modules:   pkgIdx.modules,
		Partials:  pkgIdx.partials,
		CreatedAt: pkgIdx.createdAt,
		UpdatedAt: pkgIdx.updatedAt,
	}
}
func (pkgIdx *Packages) fromPackagesJSON(p packagesJSON) {
	pkgIdx.codeRoots = p.CodeRoots
	pkgIdx.modules = p.Modules
	pkgIdx.partials = p.Partials
	pkgIdx.createdAt = p.CreatedAt
	pkgIdx.updatedAt = p.UpdatedAt
}
