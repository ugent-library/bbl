package backoffice

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views/backoffice/organizations"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type SearchOrganizationsCtx struct {
	*ctx.Ctx
	Scope string
	Opts  *bbl.SearchOpts
}

func SearchOrganizationsBinder(r *http.Request, c *ctx.Ctx) (*SearchOrganizationsCtx, error) {
	searchCtx := &SearchOrganizationsCtx{
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

type OrganizationsHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewOrganizationsHandler(repo *pgxrepo.Repo, index bbl.Index) *OrganizationsHandler {
	return &OrganizationsHandler{
		repo:  repo,
		index: index,
	}
}

func (h *OrganizationsHandler) AddRoutes(r *mux.Router, b *bind.Binder[*ctx.Ctx]) {
	searchBinder := bind.Derive(b, SearchOrganizationsBinder)

	r.Handle("/organizations", searchBinder.BindFunc(h.Search)).Methods("GET").Name("backoffice_organizations")
}

func (h *OrganizationsHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchOrganizationsCtx) error {
	hits, err := h.index.Organizations().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return organizations.Search(c.ViewCtx(), hits).Render(r.Context(), w)
}
