package main

import (
	"go/ast"
	"path"

	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/workdir"
	"github.com/muesli/termenv"
)

type toTextOptions struct {
	OutputFormat string
	Syntaxes     []outfmt.Syntax
}

func newToTextOptions(opts ...toTextOption) (opt toTextOptions) {
	opt.OutputFormat = outfmt.Format
	for _, o := range opts {
		o(&opt)
	}
	return
}

type toTextOption func(*toTextOptions)

func withOutputFormat(format string) toTextOption {
	return func(o *toTextOptions) {
		o.OutputFormat = format
	}
}

func withSyntaxes(langs ...outfmt.Syntax) toTextOption {
	return func(o *toTextOptions) {
		o.Syntaxes = append(o.Syntaxes, langs...)
	}
}
func importPathLink(pkgPath string) string {
	if outfmt.Format != outfmt.Term {
		return pkgPath
	}
	link := path.Join(outfmt.BaseURL, pkgPath)
	return termenv.Hyperlink(link, pkgPath)
}

var subs = []workdir.Sub{{
	Env:  "GOROOT",
	Path: buildCtx.GOROOT,
}, {
	Env:  "GOPATH",
	Path: buildCtx.GOPATH,
}}

func (pkg *Package) emitLocation(node ast.Node) {
	if godoc.NoLocation || godoc.Short {
		return
	}
	pos := pkg.fs.Position(node.Pos())
	if pos.Filename != "" && pos.Line > 0 {
		pkg.Printf("\n// %s +%d\n", workdir.Rel(pos.Filename, subs...), pos.Line)
	}
}
