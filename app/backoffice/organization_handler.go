package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type OrganizationHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewOrganizationHandler(repo *pgxrepo.Repo, index bbl.Index) *OrganizationHandler {
	return &OrganizationHandler{
		repo:  repo,
		index: index,
	}
}

func (h *OrganizationHandler) AddRoutes(router *mux.Router, b *bind.HandlerBinder[*ctx.Ctx]) {
}
