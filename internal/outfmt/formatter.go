package outfmt

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"

	"aslevy.com/go-doc/internal/flagvar"
	"aslevy.com/go-doc/internal/ioutil"
)

const formatEnvVar = "GODOC_FORMAT"

var (
	Format       Mode
	BaseURL      string
	GlamourStyle string
	SyntaxStyle  string

	SyntaxLang   string
	SyntaxIgnore bool
	NoSyntax     bool
)

func AddFlags(fs *flag.FlagSet) {
	Format, _ = ParseMode(os.Getenv(formatEnvVar))
	fs.Var(flagvar.Parse(&Format, ParseMode), "fmt", fmt.Sprintf("format of output: %v", Modes()))
	fs.StringVar(&BaseURL, "base-url", "https://pkg.go.dev/", "base URL for links in markdown output")
	fs.StringVar(&GlamourStyle, "theme-term", "auto", "color theme to use with -fmt=term")
	fs.StringVar(&SyntaxStyle, "theme-syntax", "monokai", "color theme for syntax highlighting with -fmt=term")
	fs.StringVar(&SyntaxLang, "syntax-lang", "go", "language to use for comment code blocks with -fmt=term|markdown")
	fs.BoolVar(&NoSyntax, "syntax-off", false, "do not use syntax highlighting anywhere")
	fs.BoolVar(&SyntaxIgnore, "syntax-ignore", false, "ignore //syntax: directives, just use -syntax-lang")
}

func IsRichMarkdown() bool {
	switch Format {
	case Markdown, Term:
		return true
	}
	return false
}

// Formatter returns output wrapped with a term formatter if -fmt=term.
func Formatter(out io.WriteCloser) io.WriteCloser {
	if Format != Term {
		return out
	}

	styleOpt := glamour.WithAutoStyle()
	if GlamourStyle != "auto" {
		styleOpt = glamour.WithStylePath(GlamourStyle)
	}

	rdr, err := glamour.NewTermRenderer(
		styleOpt,
		glamour.WithPreservedNewLines(),
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithEmoji(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		log.Printf("failed to use output format %s: %v", Format, err)
		return out
	}

	// Modify the style to make it more consistent with standard go doc
	// text output.
	if err := json.Unmarshal(stylePatchData, &rdr.AnsiOptions.Styles); err != nil {
		log.Printf("failed to use output format %s: %v", Format, err)
		return out
	}
	rdr.AnsiOptions.Styles.CodeBlock.Theme = SyntaxStyle

	return ioutil.WriteCloserFunc(rdr, func() error {
		// Close the renderer after writing all input. This triggers
		// the final conversion to the output format.
		if err := rdr.Close(); err != nil {
			return fmt.Errorf("outfmt: render: %w", err)
		}
		// Copy the formatted output to the originally given output
		// writer.
		if _, err := io.Copy(out, rdr); err != nil {
			return fmt.Errorf("outfmt: copy: %w", err)
		}
		// Finally close the original output writer.
		if err := out.Close(); err != nil {
			return fmt.Errorf("outfmt: close: %w", err)
		}
		return err
	})
}

//go:embed style-patch.json
var stylePatchData []byte
