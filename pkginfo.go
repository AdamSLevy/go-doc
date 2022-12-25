package main

import (
	"go/ast"
	"go/doc"
)

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
