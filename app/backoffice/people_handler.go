package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

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
}
