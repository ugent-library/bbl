package backoffice

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views/backoffice/people"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type SearchPeopleCtx struct {
	*ctx.Ctx
	Scope string
	Opts  *bbl.SearchOpts
}

func SearchPeopleBinder(r *http.Request, c *ctx.Ctx) (*SearchPeopleCtx, error) {
	searchCtx := &SearchPeopleCtx{
		Ctx: c,
		Opts: &bbl.SearchOpts{
			Size: 20,
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

	return searchCtx, b.Err()
}

type PeopleHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewPeopleHandler(repo *pgxrepo.Repo, index bbl.Index) *PeopleHandler {
	return &PeopleHandler{
		repo:  repo,
		index: index,
	}
}

func (h *PeopleHandler) AddRoutes(r *mux.Router, b *bind.Binder[*ctx.Ctx]) {
	searchBinder := bind.Derive(b, SearchPeopleBinder)

	r.Handle("/people", searchBinder.BindFunc(h.Search)).Methods("GET").Name("backoffice_people")
}

func (h *PeopleHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchPeopleCtx) error {
	hits, err := h.index.People().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return people.Search(c.ViewCtx(), hits).Render(r.Context(), w)
}
