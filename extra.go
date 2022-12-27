package main

import (
	"aslevy.com/go-doc/internal/astutil"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/outfmt"
)

func (pkg *Package) importDoc() {
	if godoc.NoImports || godoc.Short {
		return
	}
	defer fnewlines(&pkg.preBuf, 2)
	imports := astutil.NewPackageResolver(pkg.fs, pkg.pkg).BuildImports(pkg.imports, godoc.ShowStdlib)
	imports.Render(&pkg.preBuf)
}

const (
	delim = "```"
	begin = "\n\n" + delim + "%s\n"
	end   = "\n" + delim + "\n\n"
)

func (pb *pkgBuffer) Code() {
	if !outfmt.IsRichMarkdown() {
		return
	}
	if pb.startsWith == "" {
		pb.startsWith = "code"
	}
	if pb.inCodeBlock {
		return
	}
	pb.inCodeBlock = true
	lang := "go"
	if outfmt.NoSyntax {
		lang = "text"
	}
	pb.WriteString(delim + lang + "\n")
}
func (pb *pkgBuffer) Text() {
	if !outfmt.IsRichMarkdown() {
		return
	}
	if pb.startsWith == "" {
		pb.startsWith = "text"
	}
	if !pb.inCodeBlock {
		return
	}
	pb.inCodeBlock = false
	pb.WriteString(delim + "\n\n")
}
