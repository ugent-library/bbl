package backoffice

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views/backoffice/projects"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type SearchProjectsCtx struct {
	*ctx.Ctx
	Scope string
	Opts  *bbl.SearchOpts
}

func SearchProjectsBinder(r *http.Request, c *ctx.Ctx) (*SearchProjectsCtx, error) {
	searchCtx := &SearchProjectsCtx{
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

type ProjectsHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewProjectsHandler(repo *pgxrepo.Repo, index bbl.Index) *ProjectsHandler {
	return &ProjectsHandler{
		repo:  repo,
		index: index,
	}
}

func (h *ProjectsHandler) AddRoutes(r *mux.Router, b *bind.Binder[*ctx.Ctx]) {
	searchBinder := bind.Derive(b, SearchProjectsBinder)

	r.Handle("/projects", searchBinder.BindFunc(h.Search)).Methods("GET").Name("backoffice_projects")
}

func (h *ProjectsHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchProjectsCtx) error {
	hits, err := h.index.Projects().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return projects.Search(c.ViewCtx(), hits).Render(r.Context(), w)
}
