package pager

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"aslevy.com/go-doc/internal/executil"
	"aslevy.com/go-doc/internal/ioutil"
	"aslevy.com/go-doc/internal/open"
)

var Disabled bool

func AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Disabled, "pager-off", false, "don't use a pager")
}

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
func Pager(out io.Writer) io.WriteCloser {
	pager := getPagerEnv()
	if pager == "-" {
		Disabled = true
	}

	outNopCloser := ioutil.WriteNopCloser(out)
	if Disabled || open.Requested || !IsTTY(out) {
		return outNopCloser
	}

	pagerArgs := make([]string, 1, 2)
	pagerArgs[0] = pager
	if pager == "less" {
		pagerArgs = append(pagerArgs, "-RF")
	}

	pagerCmd, err := executil.Command(pagerArgs...)
	if err != nil {
		log.Println("pager:", err)
		return outNopCloser
	}
	pagerCmd.Stdout = out
	pagerCmd.Stderr = os.Stderr

	pagerStdin, err := pagerCmd.StdinPipe()
	if err != nil {
		log.Println("pager:", fmt.Errorf("stdin pipe: %w", err))
		return outNopCloser
	}

	if err := pagerCmd.Start(); err != nil {
		log.Println("pager:", fmt.Errorf("start: %w", err))
		return outNopCloser
	}

	return ioutil.WriteCloserFunc(pagerStdin, func() error {
		if err := pagerStdin.Close(); err != nil {
			log.Println("pager: close stdin pipe:", err)
		}
		return pagerCmd.Wait()
	})
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

// IsTTY returns true if output is a terminal, as opposed to a pipe, or some
// other buffer.
func IsTTY(out io.Writer) bool {
	f, ok := out.(*os.File)
	if !ok {
		return false
	}
	o, err := f.Stat()
	return err == nil &&
		(o.Mode()&os.ModeCharDevice) == os.ModeCharDevice
}
