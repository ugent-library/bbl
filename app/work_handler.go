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
	// router.Handle("/works/{work_id}", workCtx.Bind(h.Show)).Methods("GET").Name("show_work")
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works/new/_refresh", appCtx.Bind(h.RefreshNew)).Methods("POST").Name("refresh_new_work")
	router.Handle("/works", appCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	router.Handle("/works/{work_id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{work_id}", workCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

// func (h *WorkHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
// 	if htmx.Request(r) {
// 		return workviews.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
// 	}
// 	return workviews.ShowPage(c.ViewCtx(), c.Work).Render(r.Context(), w)
// }

func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{Kind: bbl.WorkKinds[0]}
	return workviews.Edit(c.ViewCtx(), rec).Render(r.Context(), w)
}

func (h *WorkHandler) RefreshNew(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{}
	if err := bindWorkForm(r, rec); err != nil {
		return err
	}

	return workviews.RefreshEditForm(c.ViewCtx(), rec).Render(r.Context(), w)
}

func (h *WorkHandler) Create(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{}

	if err := bindWorkForm(r, rec); err != nil {
		return err
	}

	rev := bbl.NewRev()
	rev.Add(&bbl.CreateWork{Work: rec})

	if err := h.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	rec, err := h.repo.GetWork(r.Context(), rec.ID)
	if err != nil {
		return err
	}

	return workviews.RefreshEditForm(c.ViewCtx(), rec).Render(r.Context(), w)
}

func (h *WorkHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return workviews.Edit(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	if err := bindWorkForm(r, c.Work); err != nil {
		return err
	}

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

func bindWorkForm(r *http.Request, rec *bbl.Work) error {
	var kind string
	var subKind string
	var titles []bbl.Text
	var keywords []string

	err := binder.New(r).Form().Vacuum().
		String("kind", &kind).
		String("sub_kind", &subKind).
		Each("titles", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			if attr.Val != "" {
				titles = append(titles, attr)
			}
			return true
		}).
		StringSlice("keywords", &keywords).
		Err()
	if err != nil {
		return err
	}

	rec.Kind = kind
	rec.SubKind = subKind
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	rec.Attrs.Titles = titles
	rec.Attrs.Keywords = keywords

	return nil
}
