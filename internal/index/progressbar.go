package index

import (
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

type progressBar interface {
	ChangeMax(int)
	GetMax() int
	Add(int) error
	Finish() error
	Clear() error
}

type nopProgressBar struct{}

func (nopProgressBar) ChangeMax(int) {}
func (nopProgressBar) GetMax() int   { return 0 }
func (nopProgressBar) Add(int) error { return nil }
func (nopProgressBar) Finish() error { return nil }
func (nopProgressBar) Clear() error  { return nil }

func newProgressBar(o options, total int, description string) progressBar {
	if o.disableProgressBar {
		return nopProgressBar{}
	}
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription("package index: "+description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(time.Second/3),
		progressbar.OptionShowCount(),     // show current count e.g. 3/5
		progressbar.OptionClearOnFinish(), // clear bar when done
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionUseANSICodes(true),
	)
}
