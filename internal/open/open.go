package open

import (
	"go/ast"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"

	"aslevy.com/go-doc/internal/executil"
)

var Requested bool

func IfRequested(fs *token.FileSet, node ast.Node) {
	if !Requested {
		return
	}

	pos := fs.Position(node.Pos())
	if pos.Filename == "" || pos.Line == 0 {
		log.Fatalf("failed to resolve file position for node %v", node)
	}

	args := strings.Fields(getEditorEnv())
	editorCmd, err := executil.Command(append(args, pos.Filename, "+"+strconv.Itoa(pos.Line))...)
	if err != nil {
		log.Fatal(err)
	}
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout

	if err := editorCmd.Run(); err != nil {
		log.Println(err)
	}

	os.Exit(editorCmd.ProcessState.ExitCode())
}

func getEditorEnv() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}
