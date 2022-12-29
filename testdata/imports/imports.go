package imports

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/url"
	. "os"

	"github.com/alecthomas/chroma/v2"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/muesli/termenv"
)

type E struct {
	URL url.URL
	io.Reader
	unmarshaller json.Unmarshaler

	termenv.Profile
	Renderer *glamour.TermRenderer
	config   ansi.StyleConfig
	*File
}

func (e *E) UnmarshalJSON(b []byte) error {
	var u json.Unmarshaler
	u = e.unmarshaller
	return u.UnmarshalJSON(b)
}

type e struct {
	URL url.URL
	io.Reader
	unmarshaller json.Unmarshaler

	termenv.Profile
	Renderer *glamour.TermRenderer
	config   ansi.StyleConfig

	chroma.Colour
}
