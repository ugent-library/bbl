package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	personviews "github.com/ugent-library/bbl/app/views/people"
	"github.com/ugent-library/bbl/binder"
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
	router.Handle("/people/suggest", appCtx.Bind(h.Suggest)).Methods("GET").Name("suggest_people")
}

func (h *PersonHandler) Suggest(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var query string
	var btnText string
	if err := binder.New(r).Query().String("q", &query).String("btn_text", &btnText).Err(); err != nil {
		return err
	}
	hits, err := h.index.People().Search(r.Context(), bbl.SearchArgs{Query: query, Limit: 10})
	if err != nil {
		return err
	}
	return personviews.Suggest(c.ViewCtx(), personviews.SuggestArgs{Hits: hits, BtnText: btnText}).Render(r.Context(), w)
}
