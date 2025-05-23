package app

import (
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/binder"
)

type SearchCtx struct {
	*AppCtx
	Scope string
	Opts  *bbl.SearchOpts
}

// TODO make reusable
func BindSearch(r *http.Request, appCtx *AppCtx) (*SearchCtx, error) {
	c := &SearchCtx{
		AppCtx: appCtx,
		Opts: &bbl.SearchOpts{
			Size:   20,
			Facets: []string{"kind", "status"},
		},
	}

	b := binder.New(r).
		Form().
		Vacuum().
		String("scope", &c.Scope).
		String("q", &c.Opts.Query).
		Int("size", &c.Opts.Size).
		Int("from", &c.Opts.From).
		String("cursor", &c.Opts.Cursor)
	if err := b.Err(); err != nil {
		return c, err
	}

	for _, field := range c.Opts.Facets {
		if b.Has(field) {
			c.Opts.AddFilters(bbl.Terms(field, b.GetAll(field)...))
		}
	}

	return c, b.Err()
}
