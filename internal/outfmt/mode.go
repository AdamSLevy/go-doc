package outfmt

import (
	"fmt"
	"strings"
)

const Default Mode = Text

// Mode is an output format mode.
type Mode = string

const (
	// Text is the default output format, mostly equivalent to official go
	// doc output.
	Text Mode = "text"
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
	Markdown Mode = "markdown"
	// Term renders the output of markdown with ANSI color codes and
	// hyperlinks.
	Term Mode = "term"
)

var allModes = []string{Text, Markdown, Term}

func Modes() string { return strings.Join(allModes, "|") }

func ParseMode(val string) (Mode, error) {
	val = strings.ToLower(val)
	switch val {
	case "":
		return Default, nil
	case "md":
		return Markdown, nil
	case "txt", "go":
		return Text, nil
	case "zsh":
		return Term, nil
	case Text, Markdown, Term:
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
