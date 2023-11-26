package progressbar

import (
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

type ProgressBar = progressbar.ProgressBar

func New(totalNumMods int) *progressbar.ProgressBar {
	return progressbar.NewOptions(totalNumMods,
		progressbar.OptionSetDescription("indexing modules..."),
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
