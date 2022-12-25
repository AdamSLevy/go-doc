package outfmt

import (
	_ "embed"
	"encoding/json"
	"io"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"

	"aslevy.com/go-doc/internal/ioutil"
)

var (
	Format       Mode = formatFlagDisplayDefault
	BaseURL      string
	GlamourStyle string
	SyntaxStyle  string

	SyntaxLang   string
	SyntaxIgnore bool
	NoSyntax     bool
)

func IsRichMarkdown() bool {
	switch Format {
	case RichMarkdown, Term:
		return true
	}
	return false
}

const (
	formatEnvVar             = "GODOC_FORMAT"
	formatFlagDisplayDefault = "$" + formatEnvVar + " or text"
)

func getFormat() Mode {
	mode, _ := ParseMode(os.Getenv(formatEnvVar))
	return mode
}

// Formatter returns output wrapped with a term formatter if -fmt=term.
func Formatter(output io.Writer) (io.WriteCloser, error) {
	if Format == formatFlagDisplayDefault {
		Format = getFormat()
	}

	fallback := ioutil.WriteNopCloser(output)
	if Format != Term {
		return fallback, nil
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
		return fallback, err
	}

	// Modify the style to make it more consistent with standard go doc
	// text output.
	err = json.Unmarshal(stylePatchData, &rdr.AnsiOptions.Styles)
	if err != nil {
		return fallback, err
	}
	rdr.AnsiOptions.Styles.CodeBlock.Theme = SyntaxStyle

	return ioutil.WriteCloserFunc(rdr, func() error {
		if err := rdr.Close(); err != nil {
			return err
		}
		_, err := io.Copy(output, rdr)
		return err
	}), err
}

//go:embed style-patch.json
var stylePatchData []byte
