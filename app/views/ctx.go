package views

import (
	"net/url"
	"time"

	"github.com/a-h/templ"
	"github.com/leonelquinteros/gotext"
)

type Ctx struct {
	CurrentURL *url.URL
	Route      func(name string, pairs ...string) *url.URL
	AssetPath  func(string) string
	Loc        *gotext.Locale
	// User      *biblio.User
}

func (c Ctx) SafeRoute(name string, pairs ...string) templ.SafeURL {
	return templ.SafeURL(c.Route(name, pairs...).String())
}

func (c Ctx) FormatTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}
