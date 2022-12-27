package main

import (
	"aslevy.com/go-doc/internal/astutil"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/outfmt"
)

func (pkg *Package) flushImports() {
	if godoc.NoImports || godoc.Short {
		return
	}
	imports := astutil.NewPackageResolver(pkg.fs, pkg.pkg).BuildImports(pkg.imports, godoc.ShowStdlib)
	imports.Render(pkg.writer)
}

const (
	delim = "```"
	begin = "\n\n" + delim + "%s\n"
	end   = "\n" + delim + "\n\n"
)

func (pb *pkgBuffer) Code() {
	if !outfmt.IsRichMarkdown() ||
		pb.inCodeBlock {
		return
	}
	pb.inCodeBlock = true
	lang := "go"
	if outfmt.NoSyntax {
		lang = "text"
	}
	pb.Write([]byte(delim + lang + "\n"))
}
func (pb *pkgBuffer) Text() {
	if !outfmt.IsRichMarkdown() ||
		!pb.inCodeBlock {
		return
	}
	pb.inCodeBlock = false
	pb.Write([]byte(delim + "\n\n"))
}
