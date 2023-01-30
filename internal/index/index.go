package index

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_dlog "aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/flagvar"
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
