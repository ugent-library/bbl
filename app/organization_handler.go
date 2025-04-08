package app

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ctx"
)

type OrganizationHandler struct {
	repo  *bbl.Repo
	index bbl.Index
}

func NewOrganizationHandler(repo *bbl.Repo, index bbl.Index) *OrganizationHandler {
	return &OrganizationHandler{
		repo:  repo,
		index: index,
	}
}

func (h *OrganizationHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
}
