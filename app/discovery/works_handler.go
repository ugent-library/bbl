package discovery

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views/discovery/works"
	"github.com/ugent-library/bbl/bind"
)

type SearchWorksCtx struct {
	*ctx.Ctx
	Opts *bbl.SearchOpts
}

type WorkCtx struct {
	*ctx.Ctx
	Work *bbl.Work
}

func SearchWorksBinder(r *http.Request, c *ctx.Ctx) (*SearchWorksCtx, error) {
	searchCtx := &SearchWorksCtx{
		Ctx: c,
		Opts: &bbl.SearchOpts{
			Size:   20,
			Facets: []string{"kind"},
		},
	}

	b := bind.Request(r).
		Form().
		Vacuum().
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

	searchCtx.Opts.AddFilters(bbl.Terms("status", "public"))

	return searchCtx, errors.New("STOOOOOOOOP")
}

type WorksHandler struct {
	index bbl.Index
}

func NewWorksHandler(index bbl.Index) *WorksHandler {
	return &WorksHandler{
		index: index,
	}
}

func (h *WorksHandler) AddRoutes(r *mux.Router, b *bind.Binder[*ctx.Ctx]) {
	searchBinder := bind.Derive(b, SearchWorksBinder)
	workBinder := bind.Derive(b, h.WorkBinder)

	r.Handle("/works", searchBinder.BindFunc(h.Search)).Methods("GET").Name("discovery_works")
	r.Handle("/works/{id}", workBinder.BindFunc(h.Show)).Methods("GET").Name("discovery_work")

}

func (h *WorksHandler) WorkBinder(r *http.Request, c *ctx.Ctx) (*WorkCtx, error) {
	work, err := h.index.Works().Get(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		return nil, err
	}
	return &WorkCtx{Ctx: c, Work: work}, nil
}

func (h *WorksHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchWorksCtx) error {
	hits, err := h.index.Works().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return works.Search(c.ViewCtx(), hits).Render(r.Context(), w)
}

func (h *WorksHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return works.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
}
