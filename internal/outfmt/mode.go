package outfmt

import (
	"fmt"
	"strings"
)

const Default Mode = Text

// Mode is an output format mode.
type Mode = string

const (
	// Text is the default output format, equivalent to official go doc
	// output.
	Text Mode = "text"
	// Markdown is the raw markdown rendered by the official go/doc/comment
	// package.
	Markdown Mode = "markdown"
	// HTML is the HTML rendered by the official go/doc/comment package.
	HTML Mode = "html"
	// ModeMarkdown renders markdown with "```<lang>" style code blocks.
	//
	// The <lang> for code blocks within comments is -syntax-lang, which
	// defaults to go.
	//
	// The <lang> for code blocks within doc comments can be set explicitly
	// using a //syntax: comment directive anywhere in the comment.
	//
	// The -syntax-ignore flag can be used to ignore //syntax: directives,
	// and just use -syntax-lang.
	//
	// Finally, the -no-syntax flag can be used to disable syntax
	// highlighting within code blocks entirely.
	RichMarkdown Mode = "rich-markdown"
	// Term renders the output of markdown with ANSI color codes and
	// hyperlinks.
	Term Mode = "term"
)

var allModes = []string{Text, Markdown, HTML, RichMarkdown, Term}

func Modes() string { return strings.Join(allModes, "|") }

func ParseMode(val string) (Mode, error) {
	val = strings.ToLower(val)
	switch val {
	case "":
		return Default, nil
	case "markdown-rich", "md-rich", "rich-md":
		return RichMarkdown, nil
	case "md":
		return Markdown, nil
	case "txt", "go":
		return Text, nil
	case Text, Markdown, HTML, RichMarkdown, Term:
		return val, nil
	default:
		// Use the first format with the val as its prefix to allow
		// partially typed format modes.
		for _, mode := range allModes {
			if strings.HasPrefix(mode, val) {
				return val, nil
			}
		}
		return Default, fmt.Errorf("invalid format mode %q, supported modes: %v", val, allModes)
	}
}
