package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

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

func (h *OrganizationsHandler) AddRoutes(router *mux.Router, b *bind.Binder[*ctx.Ctx]) {
}
