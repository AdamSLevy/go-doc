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
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/install"
	"aslevy.com/go-doc/internal/open"
	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/pager"
)

var debug bool

// addAllFlags to fs.
func addAllFlags(fs *flag.FlagSet) {
	fs.BoolVar(&debug, "debug", false, "Enable debug logging to stderr")

	fs.BoolVar(&godoc.NoImports, "no-imports", false, "do not show imports for referenced packages")
	fs.BoolVar(&godoc.ShowStdlib, "stdlib", false, "show imports for referenced stdlib packages")
	fs.BoolVar(&godoc.NoLocation, "no-location", false, "do not show symbol location i.e. // /path/to/circle.go +314")

	fs.BoolVar(&cache.Rebuild, "cache-rebuild", false, "Rebuild the cache")
	fs.BoolVar(&cache.Disabled, "no-cache", false, "Do not use the cache")

	fs.BoolVar(&pager.Disabled, "no-pager", false, "don't use a pager")
	fs.BoolVar(&open.Requested, "open", false, "Open the package or symbol with EDITOR")
	fs.BoolVar(&install.Requested, "install-completion", false, "Install completion files")

	fs.Var((*fmtFlag)(&outfmt.Format), "fmt", fmt.Sprintf("format of output: %v", outfmt.Modes()))
	fs.StringVar(&outfmt.BaseURL, "base-url", "https://pkg.go.dev/", "base URL for links in markdown output")
	fs.StringVar(&outfmt.GlamourStyle, "term-style", "auto", "glamour style to use with -fmt=term")
	fs.StringVar(&outfmt.SyntaxStyle, "term-style-syntax", "monokai", "code syntax style to use with -fmt=term")
	fs.StringVar(&outfmt.SyntaxLang, "syntax-lang", "go", "default '```<lang>' for comment code blocks with -fmt=rich-markdown|term")
	fs.BoolVar(&outfmt.NoSyntax, "no-syntax", false, "do not use syntax highlighting anywhere")
	fs.BoolVar(&outfmt.SyntaxIgnore, "ignore-syntax", false, "ignore //syntax: directives, just use -syntax-lang")

}

// addCompletionFlags to fs.
func addCompletionFlags(fs *flag.FlagSet, args ...string) []string {
	if len(args) == 0 || args[0] != "-complete" {
		return args
	}

	completion.Enabled = true
	fs.IntVar(&completion.Arg, "arg", 0, "The argument to complete: 1 or 2")
	fs.BoolVar(&completion.PkgsOnly, "pkgs-only", false, "Only complete packages for the first argument, ignoring the second argument")
	fs.BoolVar(&completion.ShortPath, "pkgs-short", false, "Complete packages to the shortest resolvable paths instead of full import paths")

	return args[1:]
}

type fmtFlag outfmt.Mode         // implements flag.Value
func (f fmtFlag) String() string { return string(f) }
func (f *fmtFlag) Set(s string) error {
	mode, err := outfmt.ParseMode(s)
	*f = fmtFlag(mode)
	return err
}
