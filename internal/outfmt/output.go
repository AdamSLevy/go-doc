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
	if open.Requested || completion.Requested {
		return fallback
	}

	// Set up pager and output format writers.
	pgr, err := pager.Pager(w)
	if err != nil {
		log.Println("failed to use pager:", err)
	} else {
		fallback = pgr
	}

	fmtr, err := Formatter(fallback)
	if err != nil {
		log.Printf("failed to use output format %s: %v", Format, err)
	} else {
		fallback = fmtr
	}

	return ioutil.WriteCloserFunc(fallback, func() error {
		fmtr.Close()
		return pgr.Close()
	})
}
