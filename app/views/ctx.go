package views

import (
	"net/url"
	"time"

	"github.com/leonelquinteros/gotext"
	"github.com/ugent-library/bbl"
)

type Ctx struct {
	URL                     *url.URL
	AssetPath               func(string) string
	Loc                     *gotext.Locale
	User                    *bbl.User
	CentrifugeURL           string
	GenerateCentrifugeToken func() (string, error)
}

func (c Ctx) FormatTime(t time.Time) string {
	return t.Format("2006/01/02 15:04")
}
