package backoffice

import (
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/can"
)

type SearchCtx struct {
	*ctx.Ctx
	Scope string
	Opts  *bbl.SearchOpts
}

// TODO make reusable or move to works handler
func SearchBinder(r *http.Request, c *ctx.Ctx) (*SearchCtx, error) {
	searchCtx := &SearchCtx{
		Ctx: c,
		Opts: &bbl.SearchOpts{
			Size:   20,
			Facets: []string{"kind", "status"},
		},
	}
	if can.Curate(c.User) {
		searchCtx.Scope = "curator"
	} else {
		searchCtx.Scope = "contributor"
	}

	b := bind.Request(r).
		Form().
		Vacuum().
		String("scope", &searchCtx.Scope).
		String("q", &searchCtx.Opts.Query).
		Int("size", &searchCtx.Opts.Size).
		Int("from", &searchCtx.Opts.From).
		String("cursor", &searchCtx.Opts.Cursor)
	if err := b.Err(); err != nil {
		return searchCtx, err
	}

	for _, field := range searchCtx.Opts.Facets {
		if b.Has(field) {
			searchCtx.Opts.AddFilters(bbl.Terms(field, b.GetAll(field)...))
		}
	}

	return searchCtx, b.Err()
}
