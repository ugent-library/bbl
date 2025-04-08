package app

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ctx"
)

type ProjectHandler struct {
	repo  *bbl.Repo
	index bbl.Index
}

func NewProjectHandler(repo *bbl.Repo, index bbl.Index) *ProjectHandler {
	return &ProjectHandler{
		repo:  repo,
		index: index,
	}
}

func (h *ProjectHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
}
