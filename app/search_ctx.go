package app

import (
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/binder"
)

type SearchCtx struct {
	*AppCtx
	SearchOpts *bbl.SearchOpts
}

func BindSearch(r *http.Request, appCtx *AppCtx) (*SearchCtx, error) {
	c := &SearchCtx{
		AppCtx: appCtx,
		SearchOpts: &bbl.SearchOpts{
			Size: 20,
		},
	}

	b := binder.New(r).
		Query().
		Vacuum().
		String("q", &c.SearchOpts.Query).
		Int("size", &c.SearchOpts.Size).
		Int("from", &c.SearchOpts.From).
		String("cursor", &c.SearchOpts.Cursor)
	if err := b.Err(); err != nil {
		return c, err
	}

	// TODO make reusable
	c.SearchOpts.Facets = []string{"kind", "status"}
	for _, field := range c.SearchOpts.Facets {
		if b.Has(field) {
			tf := &bbl.TermsFilter{Field: field, Terms: b.GetAll(field)}
			c.SearchOpts.Filters = append(c.SearchOpts.Filters, tf)
		}
	}

	return c, b.Err()
}
