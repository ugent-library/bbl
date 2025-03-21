package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views/forms"
	workviews "github.com/ugent-library/bbl/app/views/works"
	"github.com/ugent-library/bbl/binder"
	"github.com/ugent-library/bbl/ctx"
)

type WorkHandler struct {
	repo        *bbl.Repo
	formProfile *forms.Profile
}

func NewWorkHandler(repo *bbl.Repo, formProfile *forms.Profile) *WorkHandler {
	return &WorkHandler{
		repo:        repo,
		formProfile: formProfile,
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
	return workviews.Edit(c.ViewCtx(), c.Work, h.formProfile).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var changes []*bbl.Change

	b := binder.New(r).Form().Vacuum()

	val := bbl.Conference{}

	err := b.
		String("conference.name", &val.Name).
		String("conference.location", &val.Location).
		String("conference.organizer", &val.Organizer).
		Err()

	if err != nil {
		return err
	}

	if c.Work.Conference.IsSet() && val.IsBlank() {
		changes = append(changes, bbl.DelAttr(c.Work.ID, c.Work.Conference.ID))
	} else if c.Work.Conference.IsSet() && !val.IsBlank() {
		changes = append(changes, bbl.SetAttr(c.Work.ID, c.Work.Conference.ID, val))
	} else if !val.IsBlank() {
		changes = append(changes, bbl.AddAttr(c.Work.ID, "conference", val))
	}

	// TODO handle deletes (id's not in binder values)
	err = b.
		Each("identifier", func(b *binder.Values) bool {
			var id string
			var val bbl.Code
			b.String("id", &id)
			b.String("val.scheme", &val.Scheme)
			b.String("val.code", &val.Code)
			blank := val.IsBlank()
			if id == "" && !blank {
				changes = append(changes, bbl.AddAttr(c.Work.ID, "identifier", val))
			} else if id != "" && !blank {
				changes = append(changes, bbl.SetAttr(c.Work.ID, id, val))
			} else if id != "" && blank {
				changes = append(changes, bbl.DelAttr(c.Work.ID, id))
			}
			return true
		}).
		Err()

	if err != nil {
		return err
	}

	// TODO handle deletes (id's not in binder values)
	err = b.
		Each("title", func(b *binder.Values) bool {
			var id string
			var val bbl.Text
			b.String("id", &id)
			b.String("val.lang", &val.Lang)
			b.String("val.text", &val.Text)
			blank := val.IsBlank()
			if id == "" && !blank {
				changes = append(changes, bbl.AddAttr(c.Work.ID, "title", val))
			} else if id != "" && !blank {
				changes = append(changes, bbl.SetAttr(c.Work.ID, id, val))
			} else if id != "" && blank {
				changes = append(changes, bbl.DelAttr(c.Work.ID, id))
			}
			return true
		}).
		Err()

	if err != nil {
		return err
	}

	var keywords []string
	if b.StringSlice("keyword", &keywords).Err() != nil {
		return err
	}
	for i, attr := range c.Work.Keywords {
		if i < len(keywords) {
			changes = append(changes, bbl.SetAttr(c.Work.ID, attr.ID, bbl.Code{Scheme: "other", Code: keywords[i]}))
		} else if i >= len(keywords) {
			changes = append(changes, bbl.DelAttr(c.Work.ID, attr.ID))
		}
	}
	if len(keywords) > len(c.Work.Keywords) {
		for _, code := range keywords[len(c.Work.Keywords):] {
			changes = append(changes, bbl.AddAttr(c.Work.ID, "keyword", bbl.Code{Scheme: "other", Code: code}))
		}
	}

	if err := h.repo.AddRev(r.Context(), changes); err != nil {
		return err
	}

	work, err := h.repo.GetWork(r.Context(), c.Work.ID)
	if err != nil {
		return err
	}

	return workviews.RefreshEditForm(c.ViewCtx(), work, h.formProfile).Render(r.Context(), w)
}
