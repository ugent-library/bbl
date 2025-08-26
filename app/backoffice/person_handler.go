package backoffice

import (
	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/pgxrepo"
)

type PeopleHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewPeopleHandler(repo *pgxrepo.Repo, index bbl.Index) *PeopleHandler {
	return &PeopleHandler{
		repo:  repo,
		index: index,
	}
}

func (h *PeopleHandler) AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx]) {
}
