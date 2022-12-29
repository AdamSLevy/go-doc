package astutil

import (
	"fmt"
	"go/ast"
	"io"
	"path"
	"strconv"
	"strings"

	"aslevy.com/go-doc/internal/slices"
)

// ImportDecl represents a Go import declaration.
type ImportDecl struct {
	Doc     string
	Imports []*ImportSpec
}

// ImportSpec represents a single Go import spec.
type ImportSpec struct {
	Doc        string
	GivenName  string
	ImportPath string

	canonical string
	stdlib    *bool
}

// ParseImportSpec converts an ast.ImportSpec to an ImportSpec.
func ParseImportSpec(imp *ast.ImportSpec) *ImportSpec {
	var givenName string
	if imp.Name != nil {
		givenName = imp.Name.Name
	}
	return &ImportSpec{
		GivenName:  givenName,
		ImportPath: strings.Trim(imp.Path.Value, `"`),
	}
}
func (i *ImportSpec) LocalName() string {
	if i.GivenName != "" {
		return i.GivenName
	}
	return i.CanonicalName()
}
func (i *ImportSpec) CanonicalName() string {
	if i.canonical == "" {
		i.canonical = i.canonicalName()
	}
	return i.canonical
}
func (i *ImportSpec) canonicalName() string {
	pkgPath, pkgName := path.Split(i.ImportPath)
	if pkgPath == "" {
		return pkgName
	}

	// pkgName may be a major version, e.g. "v2", in which case we want the base segment
	if strings.HasPrefix(pkgName, "v") {
		if _, err := strconv.Atoi(pkgName[1:]); err == nil {
			return path.Base(pkgPath)
		}
	}

	return pkgName
}

func (i *ImportSpec) IsStdlib() bool {
	if i.stdlib == nil {
		stdlib := i.isStdlib()
		i.stdlib = &stdlib
	}
	return *i.stdlib
}
func (i *ImportSpec) isStdlib() bool {
	return !path.IsAbs(i.ImportPath) && // absolute paths are not stdlib
		!strings.Contains(i.ImportPath, ".") // stdlib packages do not contain dots
}

func (i *ImportDecl) Add(impI *ImportSpec) {
	isStdlib := impI.IsStdlib()
	i.Imports = slices.InsertOrReplaceFunc(i.Imports, func(impJ *ImportSpec) (insert, replace bool) {
		// stdlib first
		if isStdlib != impJ.IsStdlib() {
			insert = isStdlib
			return
		}
		// sort by import path
		if impI.ImportPath != impJ.ImportPath {
			insert = impI.ImportPath < impJ.ImportPath
			return
		}
		// sort by given name
		insert = impI.GivenName <= impJ.GivenName
		// replace if same
		replace = impI.GivenName == impJ.GivenName
		return
	}, impI)
}

func (i ImportDecl) Render(w io.Writer) error {
	if len(i.Imports) == 0 {
		return nil
	}
	var (
		commentDelim, commentNewline          = "// ", "\n"
		importOpen, importIndent, importClose = "(\n", "\t", ")\n"
	)
	if i.Doc == "" {
		commentDelim, commentNewline = "", ""
	}
	if len(i.Imports) == 1 {
		importOpen, importIndent, importClose = "", "", ""
	}

	if _, err := fmt.Fprintf(w, "%s%s%simport %s", commentDelim, i.Doc, commentNewline, importOpen); err != nil {
		return err
	}
	stdlib := i.Imports[0].IsStdlib()
	for _, imp := range i.Imports {
		if stdlib && !imp.IsStdlib() {
			w.Write([]byte("\n"))
			stdlib = false
		}

		if importIndent != "" {
			w.Write([]byte(importIndent))
		}
		if err := imp.Render(w); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%s\n", importClose)
	return err
}
func (i ImportSpec) Render(w io.Writer) error {
	var (
		nameSpace, commentSpace = " ", " "
		quote                   = `"`
		commentDelim            = "// "
	)
	if i.ImportPath == "" {
		quote = ""
		commentSpace = ""
		i.GivenName = ""
	}
	if strings.HasSuffix(i.ImportPath, "/"+i.GivenName) {
		i.GivenName = ""
	}
	if i.GivenName == "" {
		nameSpace = ""
	}
	if i.Doc == "" {
		commentSpace = ""
		commentDelim = ""
	}

	_, err := fmt.Fprintf(w, "%s%s%s%s%s%s%s%s\n", i.GivenName, nameSpace, quote, i.ImportPath, quote, commentSpace, commentDelim, i.Doc)
	return err
}
