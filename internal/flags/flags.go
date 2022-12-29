// Package flags contains all flag definitions for cmd/go-doc. The flags are
// bound to global variables in their respective packages.
//
// Additionally this package provides improved argument parsing. See AddParse.
package flags

import (
	"flag"
	"fmt"

	"aslevy.com/go-doc/internal/cache"
	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/dlog"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/install"
	"aslevy.com/go-doc/internal/open"
	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/pager"
)

// addAllFlags to fs.
func addAllFlags(fs *flag.FlagSet) {
	fs.Var(doer(dlog.Enable), "debug", "enable debug logging")
	fs.Var(doer(install.Completion), "install-completion", "install files for Zsh completion")

	fs.BoolVar(&godoc.NoImports, "imports-off", false, "do not show the imports for referenced packages")
	fs.BoolVar(&godoc.ShowStdlib, "imports-stdlib", false, "show imports for referenced stdlib packages")
	fs.BoolVar(&godoc.NoLocation, "location-off", false, "do not show symbol file location i.e. // /path/to/circle.go +314")

	fs.BoolVar(&cache.Rebuild, "cache-rebuild", false, "rebuild the cache")
	fs.BoolVar(&cache.Disabled, "cache-off", false, "do not use the cache")

	fs.BoolVar(&pager.Disabled, "pager-off", false, "don't use a pager")
	fs.BoolVar(&open.Requested, "open", false, "open the file containing the symbol with GODOC_EDITOR or EDITOR")

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
	fs.IntVar(&completion.Arg, "arg", 0, "position of arg to complete: 1, 2 or 3")
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

// doer is a flag.Value that calls itself when Set is called. It is used for
// functions which must always be called if a certain flag is set. If the flag
// is provided more than once, the function is called multiple times. The
// function must not rely on any other flag values because there is no
// guarantee that they have been set.
type doer func()                 // implements flag.Value
func (do doer) String() string   { return "" }
func (do doer) Set(string) error { do(); return nil }
func (do doer) IsBoolFlag() bool { return true }
