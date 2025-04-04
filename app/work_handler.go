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
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works/new/_refresh", appCtx.Bind(h.RefreshNew)).Methods("POST").Name("refresh_new_work")
	router.Handle("/works", appCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	router.Handle("/works/_add_abstract", appCtx.Bind(h.AddAbstract)).Methods("GET").Name("work_add_abstract")
	router.Handle("/works/_abstract", appCtx.Bind(h.Abstract)).Methods("POST").Name("work_abstract")
	router.Handle("/works/_contributor", appCtx.Bind(h.Contributor)).Methods("POST").Name("work_contributor")
	router.Handle("/works/{id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{id}/edit/_refresh", workCtx.Bind(h.RefreshEdit)).Methods("POST").Name("refresh_edit_work")
	router.Handle("/works/{id}", workCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{Kind: bbl.WorkKinds[0]}
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}
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

	rec.ID = h.repo.NewID()

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

func (h *WorkHandler) RefreshEdit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	if err := bindWorkForm(r, c.Work); err != nil {
		return err
	}

	return workviews.RefreshEditForm(c.ViewCtx(), c.Work).Render(r.Context(), w)
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

func (h *WorkHandler) AddAbstract(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	return workviews.AddAbstractDialog(c.ViewCtx(), bbl.Text{}).Render(r.Context(), w)
}

func (h *WorkHandler) Abstract(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var text bbl.Text
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		String("val", &text.Val).
		String("lang", &text.Lang).
		Err()
	if err != nil {
		return err
	}

	return workviews.AbstractField(c.ViewCtx(), idx, text).Render(r.Context(), w)
}

func (h *WorkHandler) Contributor(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var personID string
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	var con bbl.WorkContributor

	if personID != "" {
		p, err := h.repo.GetPerson(r.Context(), personID)
		if err != nil {
			return err
		}
		con.PersonID = p.ID
		con.Person = p
	}

	return workviews.ContributorField(c.ViewCtx(), idx, con).Render(r.Context(), w)
}

func bindWorkForm(r *http.Request, rec *bbl.Work) error {
	var kind string
	var subKind string
	var identifiers []bbl.Code
	var titles []bbl.Text
	var abstracts []bbl.Text
	var laySummaries []bbl.Text
	var keywords []string
	var conference bbl.Conference
	var contributors []bbl.WorkContributor

	err := binder.New(r).Form().Vacuum().
		String("kind", &kind).
		String("sub_kind", &subKind).
		Each("identifiers", func(b *binder.Values) bool {
			var attr bbl.Code
			b.String("scheme", &attr.Scheme)
			b.String("val", &attr.Val)
			if attr.Val != "" {
				identifiers = append(identifiers, attr)
			}
			return true
		}).
		Each("titles", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			if attr.Val != "" {
				titles = append(titles, attr)
			}
			return true
		}).
		Each("abstracts", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			if attr.Val != "" {
				abstracts = append(abstracts, attr)
			}
			return true
		}).
		Each("lay_summaries", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			if attr.Val != "" {
				laySummaries = append(laySummaries, attr)
			}
			return true
		}).
		StringSlice("keywords", &keywords).
		String("conference.name", &conference.Name).
		String("conference.organizer", &conference.Organizer).
		String("conference.location", &conference.Location).
		Each("contributors", func(b *binder.Values) bool {
			var con bbl.WorkContributor
			b.String("id", &con.ID)
			b.String("attrs.name", &con.Attrs.Name)
			b.String("attrs.given_name", &con.Attrs.GivenName)
			b.String("attrs.middle_name", &con.Attrs.MiddleName)
			b.String("attrs.family_name", &con.Attrs.FamilyName)
			b.String("person_id", &con.PersonID)
			contributors = append(contributors, con)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	rec.Kind = kind
	rec.SubKind = subKind
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	rec.Attrs.Identifiers = identifiers
	rec.Attrs.Titles = titles
	rec.Attrs.Abstracts = abstracts
	rec.Attrs.LaySummaries = laySummaries
	rec.Attrs.Keywords = keywords
	rec.Attrs.Conference = conference

	rec.Contributors = contributors

	return nil
}
