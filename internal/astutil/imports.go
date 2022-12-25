package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
)

type (
	// FileImports maps file names to the imports in that file.
	//
	// FileName is a type alias for a string.
	FileImports map[FileName]Imports
	// FileName is a type alias to clarify the meaning of the FileImports
	// map key.
	FileName = string

	// Imports maps package names to import paths.
	//
	// PackageName is a type aliases for string.
	Imports map[PackageName]*ImportSpec
	// PackageName is a type alias to clarify the meaning of the Imports
	// map key.
	PackageName = string
)

func (files FileImports) Add(fileName, pkgName string, imp *ImportSpec) {
	if imports, ok := files[fileName]; ok {
		imports.Add(pkgName, imp)
		return
	}
	files[fileName] = Imports{pkgName: imp}
}
func (imports Imports) Add(pkgName string, imp *ImportSpec) bool {
	if _, ok := imports[pkgName]; ok {
		return false
	}
	imports[pkgName] = imp
	return true
}

func (files FileImports) Imports(fileName string) Imports {
	imports := files[fileName]
	if imports == nil {
		imports = make(Imports)
		files[fileName] = imports
	}
	return imports
}
func (imports Imports) Import(pkgName string) *ImportSpec {
	return imports[pkgName]
}

// PackageResolver resolves import paths for package references.
//
// Because packages can have different names in different files,
// PackageResolver.ImportPath accepts a token.Pos with the package reference.
//
// PackageResolver.ImportPath searches the ast.File corresponding to the
// token.Pos for the ImportSpec with the given package name and stops when it
// finds the first match, returning the import path.
//
// As files and imports are searched, they are cached in a FileImports map.
// Subsequent calls to ImportPath with a token.Pos corresponding to previously
// searched files may be resolved from the cache. Otherwise the search
// continues where it last left off for that file.
//
// An non-nil error is returned if no corresponding file or import can be
// found.
type PackageResolver struct {
	fs    *token.FileSet
	pkg   *ast.Package
	cache FileImports
}

func NewPackageResolver(fs *token.FileSet, pkg *ast.Package) *PackageResolver {
	return &PackageResolver{
		fs:    fs,
		pkg:   pkg,
		cache: make(FileImports, len(pkg.Files)),
	}
}

// Resolve returns the ImportSpec for the package referenced at the given pos.
// If the package reference is not found, an error is returned.
//
// See PackageResolver for more details.
func (r *PackageResolver) Resolve(pkgRef string, pos token.Pos) (*ImportSpec, error) {
	fileName := r.fs.Position(pos).Filename
	if fileName == "" {
		return nil, fmt.Errorf("no filename for position %v", pos)
	}

	cache := r.cache.Imports(fileName)
	if imp := cache.Import(pkgRef); imp != nil {
		return imp, nil
	}

	file, ok := r.pkg.Files[fileName]
	if !ok {
		return nil, fmt.Errorf("no file for filename %q", fileName)
	}

	for _, imp := range file.Imports[len(cache):] {
		if imp == nil || imp.Path == nil || imp.Path.Value == "" {
			// This should not happen, but we check to be safe.
			continue
		}

		importPath := strings.Trim(imp.Path.Value, `"`)

		spec := ImportSpec{
			Path: importPath,
		}
		var pkgName string
		if imp.Name != nil {
			pkgName = imp.Name.Name
			spec.Name = pkgName
		} else {
			pkgName = filepath.Base(importPath)
		}

		cache.Add(pkgName, &spec)

		if pkgName == pkgRef {
			return &spec, nil
		}
	}
	return nil, fmt.Errorf("no import spec for %s in file %q", pkgRef, fileName)
}

func (r *PackageResolver) BuildImports(pkgRefs PackageReferences, includeStdlib bool) *ImportDecl {
	imports := ImportDecl{
		Imports: make([]*ImportSpec, 0, len(pkgRefs)),
	}

	// imported and conflicts allows us to detect if there are any package
	// name conflicts within the imports.
	imported := make(Imports, len(pkgRefs))
	var conflicts bool

	errs := make([]error, 0, len(pkgRefs))
	// For each package reference...
	for pkgRef, posies := range pkgRefs {
		for _, pos := range posies {
			// ...resolve the import path...
			imp, err := r.Resolve(pkgRef, pos)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// Select the correct slice to insert into.
			if !includeStdlib && imp.IsStdlib() {
				continue
			}
			imports.Add(imp)

			conflicts = conflicts || !imported.Add(pkgRef, imp)
		}
	}

	if conflicts {
		imports.Doc = "WARNING: The following imports contain conflicting package names."
	}

	return &imports
}
