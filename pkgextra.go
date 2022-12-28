package main

import (
	"go/ast"
	"go/doc"
	"log"
	"path"

	"aslevy.com/go-doc/internal/astutil"
	"aslevy.com/go-doc/internal/godoc"
	"aslevy.com/go-doc/internal/outfmt"
	"aslevy.com/go-doc/internal/workdir"
	"github.com/muesli/termenv"
)

// filterNodeDoc is called by Package.emit prior to rendering a node with
// [format.Node] in order to filter out unwanted documentation. The filtered
// doc is returned, if any.
//
// This prevents the node's docs from appearing redundantly as a comment above
// the rendered node, since they are shown as formatted text below the rendered
// node. This only affects [*ast.FuncDecl] and [*ast.GenDecl] nodes, otherwise
// it is a nop and doc is nil.
//
// In official go doc this is not necessary because the [doc.Package] is built
// without [doc.PreserveAST] unless the -src flag is given, in which case the
// comment is rendered as a comment, and should not be nilled out.
//
// However in this fork, the doc.Package is almost always built with
// doc.PreserveAST because it is the only way to get the comments with the
// `//syntax:` directives. Otherwise such directives are stripped from the
// comments collected by the doc.Package.
func filterNodeDoc(node any) (doc *ast.CommentGroup) {
	switch decl := node.(type) {
	case *ast.FuncDecl:
		doc = decl.Doc
		decl.Doc = nil
		decl.Body = nil
	case *ast.GenDecl:
		doc = decl.Doc
		decl.Doc = nil
	}
	return
}

var subs = []workdir.Sub{{
	Env:  "GOROOT",
	Path: buildCtx.GOROOT,
}, {
	Env:  "GOPATH",
	Path: buildCtx.GOPATH,
}}

func (pkg *Package) emitLocation(node ast.Node) {
	if godoc.NoLocation || short || showAll {
		return
	}
	pos := pkg.fs.Position(node.Pos())
	if pos.Filename != "" && pos.Line > 0 {
		pkg.newlines(1)
		pkg.Printf("// %s +%d\n", workdir.Rel(pos.Filename, subs...), pos.Line)
	}
}

func (pkg *Package) flushImports() {
	_, err := pkg.writer.Write(pkg.buf.Next(pkg.insertImports))
	if err != nil {
		log.Fatal(err)
	}
	if godoc.NoImports {
		return
	}
	astutil.NewPackageResolver(pkg.fs, pkg.pkg).
		BuildImports(pkg.imports, godoc.ShowStdlib).
		Render(pkg.writer)
}

const codeDelim = "```"

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
	pb.Buffer.Write([]byte(codeDelim + lang + "\n"))
}
func (pb *pkgBuffer) Text() {
	if !outfmt.IsRichMarkdown() ||
		!pb.inCodeBlock {
		return
	}
	pb.inCodeBlock = false
	pb.Buffer.Write([]byte(codeDelim + "\n\n"))
}

func importPathLink(pkgPath string) string {
	if outfmt.Format != outfmt.Term &&
		outfmt.BaseURL != "" {
		return pkgPath
	}
	link := path.Join(outfmt.BaseURL, pkgPath)
	return termenv.Hyperlink(link, pkgPath)
}

func (pkg *Package) Doc() *doc.Package { return pkg.doc }

// OneLineNode returns a one-line summary of the given input node.
//
// If no non-empty valName is given, the summary will be of the first exported
// value in the node, if any exist, and otherwise the empty string.
//
// If a non-empty valName is given and the node is an *ast.GenDecl, the summary
// will be of the value (const or var) with that name. This allows completion
// to render one line summaries for values that don't come first in a value
// declaration.
//
// Only the first valName is considered.
func (pkg *Package) OneLineNode(node ast.Node, name ...string) string {
	return pkg.oneLineNode(node, name...)
}
func (pkg *Package) FindTypeSpec(decl *ast.GenDecl, symbol string) *ast.TypeSpec {
	return pkg.findTypeSpec(decl, symbol)
}
func (pkg *Package) IsTypedValue(value *doc.Value) bool { return pkg.typedValue[value] }
func (pkg *Package) IsConstructor(fnc *doc.Func) bool   { return pkg.constructor[fnc] }

type OneLineNodeOption func(*OneLineNodeOptions)
type OneLineNodeOptions struct {
	valueName string
	pkgRefs   *astutil.PackageReferences
}

func WithValueName(name string) OneLineNodeOption {
	return func(o *OneLineNodeOptions) {
		o.valueName = name
	}
}
func WithPackageReferences(pkgRefs *astutil.PackageReferences) OneLineNodeOption {
	return func(o *OneLineNodeOptions) {
		o.pkgRefs = pkgRefs
	}
}
