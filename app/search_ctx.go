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

	// TODO make reusable
	c.SearchOpts.Facets = []string{"kind", "status"}
	c.SearchOpts.Filters = b.Select("kind", "status")

	return c, b.Err()
}
