package index

import (
	"os"

	"github.com/schollz/progressbar/v3"

	"aslevy.com/go-doc/internal/outfmt"
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
	if o.disableProgressBar || outfmt.Format != outfmt.Term {
		return nopProgressBar{}
	}
	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription("package index: "+description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowCount(),               // show current count e.g. 3/5
		progressbar.OptionSetRenderBlankState(true), // render at 0%
		progressbar.OptionClearOnFinish(),           // clear bar when done
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionEnableColorCodes(true),
	)
}
