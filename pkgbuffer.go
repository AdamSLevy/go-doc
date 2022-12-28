package main

import (
	"log"

	"aslevy.com/go-doc/internal/astutil"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/outfmt"
)

func (pkg *Package) flushImports() {
	_, err := pkg.writer.Write(pkg.buf.Next(pkg.insertImports))
	if err != nil {
		log.Fatal(err)
	}
	if godoc.NoImports || short {
		return
	}
	astutil.NewPackageResolver(pkg.fs, pkg.pkg).
		BuildImports(pkg.imports, godoc.ShowStdlib).
		Render(pkg.writer)
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
