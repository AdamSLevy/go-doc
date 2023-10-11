package index

import (
	"fmt"
	"strings"
	"time"
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

	dbPath string

	goRootDir     string
	goModCacheDir string
	mainModDir    string
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

func WithDBPath(dbPath string) Option {
	return func(o *options) {
		o.dbPath = dbPath
	}
}
