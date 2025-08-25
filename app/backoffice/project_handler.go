package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type ProjectHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewProjectHandler(repo *pgxrepo.Repo, index bbl.Index) *ProjectHandler {
	return &ProjectHandler{
		repo:  repo,
		index: index,
	}
}

func (h *ProjectHandler) AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx]) {
}
