package astutil

import (
	"fmt"
	"io"
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
	Doc    string
	Name   string
	Path   string
	stdlib *bool
}

func (i *ImportSpec) IsStdlib() bool {
	if i.stdlib == nil {
		stdlib := isStdlib(i.Path)
		i.stdlib = &stdlib
	}
	return *i.stdlib
}
func isStdlib(importPath string) bool {
	slash := strings.Index(importPath+"/", "/")
	return !strings.Contains(importPath[:slash], ".")
}

func (i *ImportDecl) Add(impI *ImportSpec) {
	i.Imports = slices.InsertOrReplaceFunc(i.Imports, func(impJ *ImportSpec) (bool, bool) {
		// Stdlib imports come before non-stdlib imports.
		if impI.IsStdlib() && !impJ.IsStdlib() {
			return true, false
		}
		// Imports are sorted by path, and then name.
		pathNameI := impI.Path + impI.Name
		pathNameJ := impJ.Path + impJ.Name
		return pathNameI <= pathNameJ, pathNameI == pathNameJ
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
	if i.Path == "" {
		quote = ""
		commentSpace = ""
		i.Name = ""
	}
	if strings.HasSuffix(i.Path, "/"+i.Name) {
		i.Name = ""
	}
	if i.Name == "" {
		nameSpace = ""
	}
	if i.Doc == "" {
		commentSpace = ""
		commentDelim = ""
	}

	_, err := fmt.Fprintf(w, "%s%s%s%s%s%s%s%s\n", i.Name, nameSpace, quote, i.Path, quote, commentSpace, commentDelim, i.Doc)
	return err
}
