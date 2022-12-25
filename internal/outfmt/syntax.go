package outfmt

import (
	"go/ast"
	"strconv"
	"strings"

	"aslevy.com/go-doc/internal/slices"
)

type Syntax struct {
	ID   int
	Lang string
}

func ParseSyntaxDirectives(doc *ast.CommentGroup) (langs []Syntax) {
	if doc == nil || !IsRichMarkdown() || SyntaxIgnore || NoSyntax {
		return
	}
	const directive = "//syntax:"
	lastID := -1
	for _, comment := range doc.List {
		if !strings.HasPrefix(comment.Text, directive) {
			continue
		}
		specs := strings.Fields(comment.Text[len(directive):])
		for _, spec := range specs {
			id, lang := parseSpec(spec)
			if lang == "" {
				continue
			}
			if id < 0 {
				id = lastID + 1
			}
			lastID = id
			if lang == "-" {
				lang = ""
			}
			lastID = id
			langs = slices.InsertOrReplaceFunc(langs,
				func(stx Syntax) (bool, bool) {
					return stx.ID >= id, stx.ID == id
				},
				Syntax{
					ID:   id,
					Lang: lang,
				},
			)
		}
	}
	return
}

// parseSpec returns the 0-based id and lang parsed from a single syntax
// directive spec. The spec can be in one of the following forms:
//
//  <lang>
//  <id>:<lang>
//
// The <id> is the 1-based index of a code block appearing in the corresponding
// ast.CommentGroup. The index refers to the number of code blocks in the
// comment.
//
// If <id> is less than 1 or greater than the number of code blocks in the
func parseSpec(spec string) (id int, lang string) {
	langOrID, lang, found := strings.Cut(spec, ":")
	if !found {
		return -1, langOrID
	}
	// By using only 8 bits, we limit the number of code blocks to 255.
	id64, err := strconv.ParseUint(langOrID, 10, 8)
	if err != nil || id64 < 1 {
		return -1, ""
	}
	return int(id64) - 1, lang
}
