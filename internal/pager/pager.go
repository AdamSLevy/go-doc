package pager

import (
	"fmt"
	"io"
	"os"
	"strings"

	"aslevy.com/go-doc/internal/executil"
	"aslevy.com/go-doc/internal/ioutil"
	"aslevy.com/go-doc/internal/open"
)

var Disabled bool

// Pager execs a pager and sets its stdout to the given output. A pipe to the
// pager's stdin is returned. The caller should write to the pipe and close it
// when done.
//
// This should be called after AddFlagsPager and flag.FlagSet.Parse have been
// called.
//
// The returned io.WriteCloser will never be nil and will always be valid, even
// if the error is not nil.
//
// If the -no-pager flag is set, or if the given output is not a TTY, the
// output is returned directly.
//
// The pager is determined by the environment variables GODOC_PAGER, if set,
// else PAGER. If no pager is set in the environment 'less -R' is used.
//
// If the pager fails to start, the output is returned directly with the error.
func Pager(output io.Writer) (io.WriteCloser, error) {
	fallback := ioutil.WriteNopCloser(output)
	if Disabled || !isTTY(output) || open.Requested {
		return fallback, nil
	}

	pager := getPagerEnv()
	if pager == "-" {
		Disabled = true
		return fallback, nil
	}

	pagerCmd, err := executil.Command(getPagerEnv())
	if err != nil {
		return fallback, err
	}
	pagerCmd.Stdout = output
	pagerCmd.Stderr = os.Stderr

	pagerStdin, err := pagerCmd.StdinPipe()
	if err != nil {
		return fallback, fmt.Errorf("failed to obtain stdin pipe for pager: %w", err)
	}

	if err := pagerCmd.Start(); err != nil {
		return fallback, fmt.Errorf("failed to start pager: %w", err)
	}

	return ioutil.WriteCloserFunc(pagerStdin, func() error {
		pagerStdin.Close()
		return pagerCmd.Wait()
	}), nil
}

func getPagerEnv() string {
	var envVars = []string{
		"GODOC_PAGER",
		"PAGER",
	}
	for _, envVar := range envVars {
		pager := strings.TrimSpace(os.Getenv(envVar))
		if pager != "" {
			return pager
		}
	}
	return "less"
}

// isTTY returns true if output is a terminal, as opposed to a pipe, or some
// other buffer.
func isTTY(output io.Writer) bool {
	f, ok := output.(*os.File)
	if !ok {
		return false
	}
	o, err := f.Stat()
	return err == nil &&
		(o.Mode()&os.ModeCharDevice) == os.ModeCharDevice
}
