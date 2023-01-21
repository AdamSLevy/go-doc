// Package flags contains all flag definitions for cmd/go-doc. The flags are
// bound to global variables in their respective packages.
//
// Additionally this package provides improved argument parsing. See AddParse.
package flags

import (
	"flag"
	"fmt"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/index"
	"aslevy.com/go-doc/internal/install"
	"aslevy.com/go-doc/internal/open"
	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/pager"
)

// addAllFlags to fs.
func addAllFlags(fs *flag.FlagSet) {
	fs.Var(dlog.EnableFlagValue(), "debug", "enable debug logging")
	fs.Var(index.DebugEnableFlag(), "debug-index", "enable debug logging for index")
	installCompletion := doer(install.Completion)
	fs.Var(&installCompletion, "install-completion", "install files for Zsh completion")

	fs.BoolVar(&godoc.NoImports, "imports-off", false, "do not show the imports for referenced packages")
	fs.BoolVar(&godoc.ShowStdlib, "imports-stdlib", false, "show imports for referenced stdlib packages")
	fs.BoolVar(&godoc.NoLocation, "location-off", false, "do not show symbol file location i.e. // /path/to/circle.go +314")

	fs.BoolVar(&pager.Disabled, "pager-off", false, "don't use a pager")
	fs.BoolVar(&open.Requested, "open", false, "open the file containing the symbol with GODOC_EDITOR or EDITOR")

	fs.StringVar(&index.Sync, "index", index.Auto, "cached index modes: auto, off, force, skip")

	fs.Var((*fmtFlag)(&outfmt.Format), "fmt", fmt.Sprintf("format of output: %v", outfmt.Modes()))
	fs.StringVar(&outfmt.BaseURL, "base-url", "https://pkg.go.dev/", "base URL for links in markdown output")
	fs.StringVar(&outfmt.GlamourStyle, "theme-term", "auto", "color theme to use with -fmt=term")
	fs.StringVar(&outfmt.SyntaxStyle, "theme-syntax", "monokai", "color theme for syntax highlighting with -fmt=term")
	fs.StringVar(&outfmt.SyntaxLang, "syntax-lang", "go", "language to use for comment code blocks with -fmt=term|markdown")
	fs.BoolVar(&outfmt.NoSyntax, "syntax-off", false, "do not use syntax highlighting anywhere")
	fs.BoolVar(&outfmt.SyntaxIgnore, "syntax-ignore", false, "ignore //syntax: directives, just use -syntax-lang")
}

// addCompletionFlags to fs.
func addCompletionFlags(fs *flag.FlagSet) {
	fs.IntVar(&completion.Current, "arg", 0, "position of arg to complete: 1, 2 or 3")
	fs.BoolVar(&completion.PkgsOnly, "pkgs-only", false, "do not suggest symbols, only packages")
	fs.BoolVar(&completion.ShortPath, "pkgs-short", false, "suggest the shortest unique right-partial path, instead of the full import path i.e. json instead of encoding/json")
}

type fmtFlag outfmt.Mode         // implements flag.Value
func (f fmtFlag) String() string { return string(f) }
func (f *fmtFlag) Set(s string) error {
	mode, err := outfmt.ParseMode(s)
	*f = fmtFlag(mode)
	return err
}

// doer is a function implementing a boolean flag.Value which calls itself when
// Set is called. It is used for functions which must be called if a flag is
// set like -debug and -install-completion.
//
// Since all arguments from the command line are also passed through to
// completion, we must be able to disable certain flags to prevent them from
// hijacking completion. For example if the user is typing `go doc -debug ...`
// then we don't want `go-doc -complete ... -debug ...` to print debug logs.
//
// If the flag is set to "disable" then all subsequent appearences of the flag
// are silently ignored and the function is never called. Note that since this
// is a bool flag this must be specified as a single argument with an equal
// sign, e.g. `-debug=disable`.
type doer func()                 // implements flag.Value
func (do doer) call()            { do() }
func (do doer) String() string   { return "" }
func (do doer) IsBoolFlag() bool { return true }
func (do *doer) Set(val string) error {
	if val == "disable" {
		*do = nop
	} else if val == "true" {
		do.call()
	}
	return nil
}
func nop() {}
