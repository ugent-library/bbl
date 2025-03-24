package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	workviews "github.com/ugent-library/bbl/app/views/works"
	"github.com/ugent-library/bbl/binder"
	"github.com/ugent-library/bbl/ctx"
)

type WorkHandler struct {
	repo *bbl.Repo
}

func NewWorkHandler(repo *bbl.Repo) *WorkHandler {
	return &WorkHandler{
		repo: repo,
	}
}

func (h *WorkHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
	workCtx := ctx.Derive(appCtx, BindWorkCtx(h.repo))
	// router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	// router.Handle("/works/new/_refresh", appCtx.Bind(h.RefreshNew)).Methods("POST").Name("refresh_new_work")
	// router.Handle("/works", appCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	// router.Handle("/works/{work_id}", workCtx.Bind(h.Show)).Methods("GET").Name("show_work")
	router.Handle("/works/{work_id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{work_id}", workCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

// func (h *WorkHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
// 	if htmx.Request(r) {
// 		return workviews.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
// 	}
// 	return workviews.ShowPage(c.ViewCtx(), c.Work).Render(r.Context(), w)
// }

// func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
// 	return workviews.Edit(c.ViewCtx(), workFormProfile, biblio.NewWork()).Render(r.Context(), w)
// }

// func (h *WorkHandler) RefreshNew(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
// 	f := &WorkForm{}
// 	rec := biblio.NewWork()
// 	if err := f.Bind(r, rec); err != nil {
// 		return err
// 	}

// 	return workviews.RefreshEditForm(c.ViewCtx(), workFormProfile, rec).Render(r.Context(), w)
// }

// func (h *WorkHandler) Create(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
// 	f := &WorkForm{}
// 	rec := biblio.NewWork()
// 	if err := f.Bind(r, rec); err != nil {
// 		return err
// 	}

// 	rev := biblio.NewRevision()
// 	rev.Add(biblio.NewWorkChangeset().Add(rec.Changes()...))
// 	if err := h.repo.AddRevision(r.Context(), rev); err != nil {
// 		return err
// 	}

// 	return workviews.RefreshEditForm(c.ViewCtx(), workFormProfile, rec).Render(r.Context(), w)
// }

func (h *WorkHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return workviews.Edit(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	b := binder.New(r).Form().Vacuum()

	var titles []bbl.Text
	b.Each("titles", func(b *binder.Values) bool {
		var attr bbl.Text
		b.String("lang", &attr.Lang)
		b.String("val", &attr.Val)
		if attr.Val != "" {
			titles = append(titles, attr)
		}
		return true
	})

	if err := b.Err(); err != nil {
		return err
	}
	c.Work.Attrs.Titles = titles

	var keywords []string
	if err := b.StringSlice("keywords", &keywords).Err(); err != nil {
		return err
	}
	c.Work.Attrs.Keywords = keywords

	rev := bbl.NewRev()
	rev.Add(&bbl.UpdateWork{Work: c.Work})

	if err := h.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	work, err := h.repo.GetWork(r.Context(), c.Work.ID)
	if err != nil {
		return err
	}

	return workviews.RefreshEditForm(c.ViewCtx(), work).Render(r.Context(), w)
}
