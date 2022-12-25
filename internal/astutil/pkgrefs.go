package astutil

import (
	"go/ast"
	"go/token"
)

// PackageReferences maps package names to the positions the package name found
// in the AST.
//
// Note that the package name may not be the actual name of the package being
// referenced, since imported packages can have different names in different
// files.
//
// Use the PackageResolver to resolve the package name and token.Pos to the
// actual import path.
type PackageReferences map[PackageName][]token.Pos

// FindPackageReferences finds any external package references in the AST of
// the node.
func FindPackageReferences(node ast.Node) PackageReferences {
	pkgRefs := make(PackageReferences)
	pkgRefs.Find(node)
	return pkgRefs
}

func (pkgRefs PackageReferences) Find(node ast.Node) {
	p := pkgRefFinder{pkgRefs: pkgRefs}
	ast.Walk(p, node)
}

func Merge(a, b PackageReferences) PackageReferences {
	long, short := a, b
	if len(short) > len(long) {
		long, short = short, long
	}
	for k, v := range short {
		long.Add(k, v...)
	}
	return long
}

func (pkgRefs PackageReferences) Add(pkgName string, pos ...token.Pos) {
	pkgRefs[pkgName] = append(pkgRefs[pkgName], pos...)
}

// pkgRefFinder implements ast.Visitor and collects all external
// references.
//
// External references are *ast.SelectorExprs where the X is an *ast.Ident that
// has a nil Obj.
//
// Because each file may name imports differently, we collect the token
// positions of each referenced name so that we can look up the ImportSpec from
// that specific file later.
//
// This does lead to the possibility that the same package is imported under
// two different names, or worse, that the same name is used to refer to
// different packages. In the former case, the user can disambiguate easily. In
// the latter case we will print a note pointing out that the import block has
// package name conflicts. A user will need to drill down to the specific
// symbol they are interested in to disambiguate the package name.
type pkgRefFinder struct {
	pkgRefs PackageReferences
	depth   int
}

func (p pkgRefFinder) Visit(node ast.Node) ast.Visitor {
	if node == nil || p.depth > 10 {
		return nil
	}
	switch n := node.(type) {
	case *ast.SelectorExpr:
		if pkg, ok := n.X.(*ast.Ident); ok && pkg.Obj == nil {
			p.pkgRefs.Add(pkg.Name, pkg.Pos())
		}
		// No need to descend into the selector.
		return nil
	case *ast.CommentGroup:
		// Do not descend into comments.
		return nil
	}
	p.depth++
	return p
}
