package imports

import (
	"encoding/json"
	"io"
	"net/url"

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
