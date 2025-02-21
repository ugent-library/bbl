package views

import (
	"net/url"

	"github.com/a-h/templ"
)

type Ctx struct {
	Route     func(name string, pairs ...string) *url.URL
	AssetPath func(string) string
	// User      *biblio.User
}

func (c Ctx) SafeRoute(name string, pairs ...string) templ.SafeURL {
	return templ.SafeURL(c.Route(name, pairs...).String())
}
