package open

import (
	"bytes"
	"flag"
	"go/ast"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"aslevy.com/go-doc/internal/completion"
	"aslevy.com/go-doc/internal/executil"
)

var Requested bool

func AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&Requested, "open", false, "open the file containing the symbol with GODOC_EDITOR or EDITOR")
}

func IfRequested(fs *token.FileSet, node ast.Node) {
	if !Requested || completion.Requested {
		return
	}

	switch node.(type) {
	case *ast.Package:
		log.Fatal("cannot use -open if no symbol is specified")
	}

	log.Println("opening the symbol in your editor...")
	pos := fs.Position(node.Pos())
	if pos.Filename == "" {
		log.Fatalf("failed to determine the file containing the symbol")
	}
	if pos.Line == 0 {
		log.Printf("failed to determine the line number of the symbol")
	}

	args := getEditorArgs(pos)
	editorCmd, err := executil.Command(args...)
	if err != nil {
		log.Fatal(err)
	}
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		log.Println(err)
	}

	os.Exit(editorCmd.ProcessState.ExitCode())
}

func getEditorArgs(pos token.Position) []string {
	editor := os.Getenv("GODOC_EDITOR")
	if editor != "" {
		var ok bool
		editor, ok = parseGodocEditor(editor, pos)
		if ok {
			return strings.Fields(editor)
		}
	}
	if editor == "" {
		if editor = os.Getenv("EDITOR"); editor == "" {
			log.Fatal("failed to determine editor: please set GODOC_EDITOR or EDITOR in your environment")
		}
	}
	editor = strings.TrimSpace(editor)
	line := strconv.Itoa(pos.Line)
	switch filepath.Base(editor) {
	case "vi", "vim", "nvim", "gvim":
		// vim variants use the same syntax to specify a line number
		return []string{editor, "+" + line, pos.Filename}
	default:
	}
	log.Printf("unrecognized editor: cannot jump to line %v", line)
	return []string{editor, pos.Filename}
}
func parseGodocEditor(editor string, pos token.Position) (string, bool) {
	if !strings.Contains(editor, "{{") {
		return editor, false
	}
	tmpl, err := template.New("GODOC_EDITOR").Parse(editor)
	if err != nil {
		log.Println("failed to parse the GODOC_EDITOR template:", err)
		return "", false
	}
	var buf bytes.Buffer
	if err := tmpl.Option("missingkey=error").Execute(&buf, pos); err != nil {
		log.Println("failed to execute the GODOC_EDITOR template:", err)
		return "", false
	}
	return buf.String(), true
}
