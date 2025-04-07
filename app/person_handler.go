package app

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ctx"
)

type PersonHandler struct {
	repo  *bbl.Repo
	index bbl.Index
}

func NewPersonHandler(repo *bbl.Repo, index bbl.Index) *PersonHandler {
	return &PersonHandler{
		repo:  repo,
		index: index,
	}
}

func (h *PersonHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
}
