package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type PersonHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewPersonHandler(repo *pgxrepo.Repo, index bbl.Index) *PersonHandler {
	return &PersonHandler{
		repo:  repo,
		index: index,
	}
}

func (h *PersonHandler) AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx]) {
}
