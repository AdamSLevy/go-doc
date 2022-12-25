// Package ioutil provides some I/O utility functions not offered by the
// official stdlib io or io/ioutil.
package ioutil

import (
	"io"
)

// WriteNopCloser turns w into an io.WriteCloser with a no-op Close method.
func WriteNopCloser(w io.Writer) io.WriteCloser {
	return writeCloserFunc{w, nopClose}
}
func nopClose() error { return nil }

// WriteCloserFunc turns w into an io.WriteCloser with the given closeFunc.
//
// This can be used to bundle additional cleanup logic with an io.Writer, or to
// override the Close function of an io.WriteCloser.
func WriteCloserFunc(w io.Writer, closeFunc func() error) io.WriteCloser {
	return writeCloserFunc{w, closeFunc}
}

type writeCloserFunc struct {
	io.Writer
	closeFunc func() error
}

func (w writeCloserFunc) Close() error { return w.closeFunc() }
