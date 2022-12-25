package outfmt

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/muesli/reflow/wordwrap"
)

const (
	delim  = "```"
	begin  = "\n\n" + delim + "%s\n"
	end    = "\n" + delim + "\n\n"
	indent = "    "
)

func ReformatMarkdown(data []byte, langs ...Syntax) []byte {
	data = ReformatListBlocks(data)
	data = ReformatTextBlocks(data)
	return ReformatCodeBlocks(data, langs...)
}

// codeBlocks matches simple code blocks in Markdown as rendered by
// [comment.Printer.Markdown]. Code blocks are indented by a single tab, and
// are preceeded and followed by two newlines.
//
// This regex can be viewed and better understood here:
// https://regex101.com/r/1gbLMe/2
var codeBlocks = regexp.MustCompile(`(?:\A|\n\n)((?:\t.*?\n+)+)(?:\n|\z)`)

// ReformatCodeBlocks reformats markdown code blocks from:
//
// Package or symbol comment...
//
//     func Hello() {
//           ...
//     }
//
// Into:
//
// Package or symbol comment...
//
//   ```go
//       func Hello() {
//           ...
//       }
//   ```
func ReformatCodeBlocks(data []byte, langs ...Syntax) []byte {
	var langsID, codeBlockID int
	lang := SyntaxLang
	if lang == "auto" {
		lang = "go"
	}
	if NoSyntax {
		lang = "text"
	}
	return codeBlocks.ReplaceAllFunc(data, func(block []byte) []byte {
		if langsID < len(langs) && langs[langsID].ID == codeBlockID {
			lang = langs[langsID].Lang
			langsID++
		} else if SyntaxLang == "auto" {
			lexer := lexers.Analyse(string(block))
			if lexer != nil {
				if config := lexer.Config(); config != nil {
					lang = config.Name
				}
			}
		}
		codeBlockID++

		// Replace all leading tabs with the standard indent.
		block = bytes.ReplaceAll(block, []byte("\n\t"), []byte("\n"+indent))
		// Remove all leading and trailing newlines.
		block = bytes.Trim(block, "\n")

		buf := bytes.NewBuffer(make([]byte, 0, len(block)+len(begin)-2+len(lang)+len(end)))
		buf.WriteString(fmt.Sprintf(begin, lang))
		buf.Write(block)
		buf.WriteString(end)
		return buf.Bytes()
	})
}

var textBlocks = regexp.MustCompile(`(?:\n|\A)(?:[^\s\t]+.*\n+)+`)

func ReformatTextBlocks(data []byte) []byte {
	return textBlocks.ReplaceAllFunc(data, func(block []byte) []byte {
		block = unescapeMarkdown(block)
		return wordwrap.Bytes(block, 80)
	})
}

var listBlocks = regexp.MustCompile(`(?m:^(\s\s-\s)|(\s\d+\.\s)).*\n+`)

func ReformatListBlocks(data []byte) []byte {
	return listBlocks.ReplaceAllFunc(data, func(block []byte) []byte {
		block = unescapeMarkdown(block)
		block = wordwrap.Bytes(block, 80)
		return bytes.TrimRight(
			bytes.ReplaceAll(block, []byte("\n"), []byte("\n    ")),
			" ")
	})
}

var escapedMarkdownChars = regexp.MustCompile(`\\([\[\\*_<.` + "`" + `])`)

func unescapeMarkdown(data []byte) []byte {
	return escapedMarkdownChars.ReplaceAll(data, []byte("$1"))
}
