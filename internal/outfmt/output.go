package outfmt

import (
	"io"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/ioutil"
	"aslevy.com/go-doc/internal/open"
	"aslevy.com/go-doc/internal/pager"
)

func Output(out io.Writer) io.WriteCloser {
	if open.Requested || completion.Requested {
		return ioutil.WriteNopCloser(out)
	}

	// Set up pager and output format writers.
	pgr := pager.Pager(out)
	fmtr := Formatter(pgr)

	return fmtr
}
