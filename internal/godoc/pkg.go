// Package godoc provides interfaces for APIs defined in the cmd/go-doc main
// package that are also used by the completion package.
//
// This allows us to share code with the completion package without major
// refactors of the cmd/go-doc package. Minimizing the diff makes it easier to
// merge upstream changes to the official go doc.
package godoc

import (
	"go/ast"
	"go/doc"

	"aslevy.com/go-doc/internal/astutil"
)

// Package exposes the information about a package that is needed by the
// completion package.
type PackageInfo interface {
	Doc() *doc.Package
	IsTypedValue(value *doc.Value) bool
	IsConstructor(value *doc.Func) bool

	// OneLineNode returns a one-line summary of the given input node.
	//
	// If no non-empty valName is given, the summary will be of the first
	// exported value in the node, if any exist, and otherwise the empty
	// string.
	//
	// If a non-empty valName is given and the node is an *ast.GenDecl, the
	// summary will be of the value (const or var) with that name. This
	// allows completion to render one line summaries for values that don't
	// come first in a value declaration.
	//
	// Only the first valName is considered.
	OneLineNode(node ast.Node, opts ...OneLineNodeOption) string

	// FindTypeSpec returns the ast.TypeSpec within the declaration that
	// defines the symbol. The name must match exactly.
	FindTypeSpec(decl *ast.GenDecl, symbol string) *ast.TypeSpec
}

type OneLineNodeOption func(*OneLineNodeOptions)
type OneLineNodeOptions struct {
	ValueName string
	PkgRefs   astutil.PackageReferences
}

func NewOneLineNodeOptions(opts ...OneLineNodeOption) (o OneLineNodeOptions) {
	WithOpts(opts...)(&o)
	return
}

func WithOpts(opts ...OneLineNodeOption) OneLineNodeOption {
	return func(o *OneLineNodeOptions) {
		for _, opt := range opts {
			opt(o)
		}
	}
}

func WithValueName(name string) OneLineNodeOption {
	return func(o *OneLineNodeOptions) {
		o.ValueName = name
	}
}

func WithPkgRefs(pkgRefs astutil.PackageReferences) OneLineNodeOption {
	return func(o *OneLineNodeOptions) {
		o.PkgRefs = pkgRefs
	}
}
