package outfmt

import (
	"io"
	"log"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/ioutil"
	"aslevy.com/go-doc/internal/open"
	"aslevy.com/go-doc/internal/pager"
)

func Output(w io.Writer) io.WriteCloser {
	fallback := ioutil.WriteNopCloser(w)
	if open.Requested || completion.Enabled {
		return fallback
	}

	closeFuns := make([]func() error, 0, 2)
	// Set up pager and output format writers.
	pgr, err := pager.Pager(w)
	if err != nil {
		log.Println("failed to use pager:", err)
	} else {
		closeFuns = append(closeFuns, pgr.Close)
		fallback = pgr
	}

	fmtr, err := Formatter(fallback)
	if err != nil {
		log.Println("failed to use output format %s: %w", Format, err)
	} else {
		closeFuns = append(closeFuns, fmtr.Close)
		fallback = fmtr
	}

	return ioutil.WriteCloserFunc(fallback, func() error {
		for _, closeFun := range closeFuns {
			defer closeFun()
		}
		fmtr.Close()
		return pgr.Close()
	})
}
