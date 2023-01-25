package index

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/goccy/go-json"

	_dlog "aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/flagvar"
	"aslevy.com/go-doc/internal/godoc"
)

// IMPROVEMENTS:
// - lazy loading:
//  - load/sync index only after first search
//  - load only the partials list required for the search
//    - This will require splitting the modules and each partials lists into
//      separate files.
//  - stop the search at the first result and find a way to resume the search later

const (
	SyncEnvVar            = "GODOC_INDEX_MODE"
	ResyncEnvVar          = "GODOC_INDEX_RESYNC"
	DefaultResyncInterval = 20 * time.Minute
	NoProgressBar         = "GODOC_NO_PROGRESS_BAR"
)

var (
	dlog           = _dlog.Child("index")
	Sync           = ModeAutoSync
	ResyncInterval = DefaultResyncInterval
)

func AddFlags(fs *flag.FlagSet) {
	fs.Var(dlog.EnableFlag(), "debug-index", "enable debug logging for index")
	Sync, _ = ParseMode(os.Getenv(SyncEnvVar))
	fs.Var(flagvar.Parse(&Sync, ParseMode), "index-mode", fmt.Sprintf("cached index modes: %s", modes()))
	fs.DurationVar(&ResyncInterval, "index-resync", parseResyncInterval(os.Getenv(ResyncEnvVar)), "resync index if older than this duration")
}
func parseResyncInterval(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return DefaultResyncInterval
	}
	return d
}

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

type Packages struct {
	codeRoots []godoc.PackageDir
	modules   moduleList
	partials  rightPartialIndex

	createdAt time.Time
	updatedAt time.Time

	goDocBuildSum string

	options
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
	CodeRoots     []godoc.PackageDir
	Modules       moduleList
	Partials      rightPartialIndex
	CreatedAt     time.Time
	UpdatedAt     time.Time
	BuildRevision string
}

func (pkgIdx Packages) toPackagesJSON() packagesJSON {
	return packagesJSON{
		CodeRoots:     pkgIdx.codeRoots,
		Modules:       pkgIdx.modules,
		Partials:      pkgIdx.partials,
		CreatedAt:     pkgIdx.createdAt,
		UpdatedAt:     pkgIdx.updatedAt,
		BuildRevision: getBuildRevision(),
	}
}
func (pkgIdx *Packages) fromPackagesJSON(p packagesJSON) {
	pkgIdx.codeRoots = p.CodeRoots
	pkgIdx.modules = p.Modules
	pkgIdx.partials = p.Partials
	pkgIdx.createdAt = p.CreatedAt
	pkgIdx.updatedAt = p.UpdatedAt

	rev := getBuildRevision()
	if rev != "" && rev != p.BuildRevision {
		pkgIdx.options.mode = ModeForceSync
	}
}

func getBuildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}
