package views

import (
	"net/url"
	"time"

	"github.com/a-h/templ"
	"github.com/leonelquinteros/gotext"
	"github.com/ugent-library/bbl"
)

type Ctx struct {
	URL       *url.URL
	RouteName string
	Route     func(string, ...any) *url.URL
	AssetPath func(string) string
	SSEPath   func() string
	Loc       *gotext.Locale
	User      *bbl.User
}

func (c Ctx) SafeRoute(name string, params ...any) templ.SafeURL {
	return templ.SafeURL(c.Route(name, params...).String())
}

func (c Ctx) FormatTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}
