package index

import (
	"flag"
	"fmt"
	"os"
	"time"

	_dlog "aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/flagvar"
)

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
	debugDesc := "enable debug logging for index"
	fs.Var(dlog.EnableFlag(), "debug-index", debugDesc)
	fs.Var(dlogSearch.EnableFlag(), "debug-index-search", debugDesc+" search")
	fs.Var(dlogSync.EnableFlag(), "debug-index-sync", debugDesc+" sync")

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
