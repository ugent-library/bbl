package app

import (
	"fmt"
	"net/http"
	"slices"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	workviews "github.com/ugent-library/bbl/app/views/works"
	"github.com/ugent-library/bbl/binder"
	"github.com/ugent-library/bbl/ctx"
	"github.com/ugent-library/bbl/pgxrepo"
)

type WorkHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewWorkHandler(repo *pgxrepo.Repo, index bbl.Index) *WorkHandler {
	return &WorkHandler{
		repo:  repo,
		index: index,
	}
}

func (h *WorkHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
	workCtx := ctx.Derive(appCtx, BindWorkCtx(h.repo))
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works/new/_refresh", appCtx.Bind(h.RefreshNew)).Methods("POST").Name("refresh_new_work")
	router.Handle("/works", appCtx.Bind(h.Create)).Methods("POST").Name("create_work")

	router.Handle("/works/_add_abstract", appCtx.Bind(h.AddAbstract)).Methods("POST").Name("work_add_abstract")
	router.Handle("/works/_edit_abstract", appCtx.Bind(h.EditAbstract)).Methods("POST").Name("work_edit_abstract")
	router.Handle("/works/_remove_abstract", appCtx.Bind(h.RemoveAbstract)).Methods("POST").Name("work_remove_abstract")
	router.Handle("/works/_add_lay_summary", appCtx.Bind(h.AddLaySummary)).Methods("POST").Name("work_add_lay_summary")
	router.Handle("/works/_edit_lay_summary", appCtx.Bind(h.EditLaySummary)).Methods("POST").Name("work_edit_lay_summary")
	router.Handle("/works/_remove_lay_summary", appCtx.Bind(h.RemoveLaySummary)).Methods("POST").Name("work_remove_lay_summary")
	router.Handle("/works/_suggest_contributors", appCtx.Bind(h.SuggestContributors)).Methods("GET").Name("work_suggest_contributors")
	router.Handle("/works/_add_contributor", appCtx.Bind(h.AddContributor)).Methods("POST").Name("work_add_contributor")
	router.Handle("/works/_edit_contributor", appCtx.Bind(h.EditContributor)).Methods("POST").Name("work_edit_contributor")
	router.Handle("/works/_remove_contributor", appCtx.Bind(h.RemoveContributor)).Methods("POST").Name("work_remove_contributor")

	router.Handle("/works/{id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{id}/edit/_refresh", workCtx.Bind(h.RefreshEdit)).Methods("POST").Name("refresh_edit_work")
	router.Handle("/works/{id}", workCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{Kind: bbl.WorkKinds[0]}
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	route := c.Route("create_work")

	return workviews.Edit(c.ViewCtx(), rec, route).Render(r.Context(), w)
}

func (h *WorkHandler) RefreshNew(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{}
	if _, err := bindWorkForm(r, rec); err != nil {
		return err
	}

	return workviews.RefreshForm(c.ViewCtx(), rec).Render(r.Context(), w)
}

func (h *WorkHandler) Create(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{}

	refresh, err := bindWorkForm(r, rec)
	if err != nil {
		return err
	}

	route := c.Route("create_work")

	if refresh != "" {
		return workviews.Form(c.ViewCtx(), rec, route).Render(r.Context(), w)
	}

	rec.ID = bbl.NewID()

	rev := bbl.NewRev()
	rev.Add(&bbl.CreateWork{Work: rec})

	if err = h.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	rec, err = h.repo.GetWork(r.Context(), rec.ID)
	if err != nil {
		return err
	}

	return workviews.RefreshForm(c.ViewCtx(), rec).Render(r.Context(), w)
}

func (h *WorkHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	route := c.Route("create_work", "id", c.Work.ID)

	return workviews.Edit(c.ViewCtx(), c.Work, route).Render(r.Context(), w)
}

func (h *WorkHandler) RefreshEdit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	if _, err := bindWorkForm(r, c.Work); err != nil {
		return err
	}

	return workviews.RefreshForm(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	if _, err := bindWorkForm(r, c.Work); err != nil {
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

	return workviews.RefreshForm(c.ViewCtx(), work).Render(r.Context(), w)
}

func (h *WorkHandler) AddAbstract(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var text bbl.Text
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		String("lang", &text.Lang).
		String("val", &text.Val).
		Each("abstracts", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	texts = append(texts, text)

	return workviews.AbstractsField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) EditAbstract(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var text bbl.Text
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		String("lang", &text.Lang).
		String("val", &text.Val).
		Each("abstracts", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	if idx >= 0 && idx < len(texts) {
		texts[idx] = text
	}

	return workviews.AbstractsField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) RemoveAbstract(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		Each("abstracts", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	if idx >= 0 && idx < len(texts) {
		texts = append(texts[:idx], texts[idx+1:]...)
	}

	return workviews.AbstractsField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) AddLaySummary(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var text bbl.Text
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		String("lang", &text.Lang).
		String("val", &text.Val).
		Each("lay_summaries", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	texts = append(texts, text)

	return workviews.LaySummariesField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) EditLaySummary(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var text bbl.Text
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		String("lang", &text.Lang).
		String("val", &text.Val).
		Each("lay_summaries", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	if idx >= 0 && idx < len(texts) {
		texts[idx] = text
	}

	return workviews.LaySummariesField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) RemoveLaySummary(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var texts []bbl.Text
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		Each("lay_summaries", func(b *binder.Values) bool {
			var attr bbl.Text
			b.String("lang", &attr.Lang)
			b.String("val", &attr.Val)
			texts = append(texts, attr)
			return true
		}).
		Err()
	if err != nil {
		return err
	}

	if idx >= 0 && idx < len(texts) {
		texts = append(texts[:idx], texts[idx+1:]...)
	}

	return workviews.LaySummariesField(c.ViewCtx(), texts).Render(r.Context(), w)
}

func (h *WorkHandler) SuggestContributors(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var query string
	var idx int = -1
	if err := binder.New(r).Query().String("q", &query).Int("idx", &idx).Err(); err != nil {
		return err
	}
	hits, err := h.index.People().Search(r.Context(), bbl.SearchOpts{Query: query, Limit: 10})
	if err != nil {
		return err
	}
	return workviews.ContributorSuggestions(c.ViewCtx(), hits, idx).Render(r.Context(), w)
}

func (h *WorkHandler) AddContributor(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var personID string
	var contributors []bbl.WorkContributor
	err := binder.New(r).Form().Vacuum().
		String("person_id", &personID).
		Each("contributors", func(b *binder.Values) bool {
			var con bbl.WorkContributor
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

	for i, con := range contributors {
		if con.PersonID != "" {
			p, err := h.repo.GetPerson(r.Context(), con.PersonID)
			if err != nil {
				return err
			}
			contributors[i].Person = p
		}
	}

	if personID != "" {
		p, err := h.repo.GetPerson(r.Context(), personID)
		if err != nil {
			return err
		}
		contributors = append(contributors, bbl.WorkContributor{
			PersonID: p.ID,
			Person:   p,
		})
	}

	return workviews.ContributorsField(c.ViewCtx(), contributors).Render(r.Context(), w)
}

func (h *WorkHandler) EditContributor(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var personID string
	var contributors []bbl.WorkContributor
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		String("person_id", &personID).
		Each("contributors", func(b *binder.Values) bool {
			var con bbl.WorkContributor
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

	if idx >= 0 && idx < len(contributors) {
		if personID != "" {
			p, err := h.repo.GetPerson(r.Context(), personID)
			if err != nil {
				return err
			}
			contributors[idx].Person = p
		}
	}

	for i, con := range contributors {
		if i != idx && con.PersonID != "" {
			p, err := h.repo.GetPerson(r.Context(), con.PersonID)
			if err != nil {
				return err
			}
			contributors[i].Person = p
		}
	}

	return workviews.ContributorsField(c.ViewCtx(), contributors).Render(r.Context(), w)
}

func (h *WorkHandler) RemoveContributor(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var idx int
	var contributors []bbl.WorkContributor
	err := binder.New(r).Form().Vacuum().
		Int("idx", &idx).
		Each("contributors", func(b *binder.Values) bool {
			var con bbl.WorkContributor
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

	if idx >= 0 && idx < len(contributors) {
		contributors = append(contributors[:idx], contributors[idx+1:]...)
	}

	for i, con := range contributors {
		if con.PersonID != "" {
			p, err := h.repo.GetPerson(r.Context(), con.PersonID)
			if err != nil {
				return err
			}
			contributors[i].Person = p
		}
	}

	return workviews.ContributorsField(c.ViewCtx(), contributors).Render(r.Context(), w)
}

func bindWorkForm(r *http.Request, rec *bbl.Work) (string, error) {
	var kind string
	var subKind string
	var identifiers []bbl.Code
	var titles []bbl.Text
	var abstracts []bbl.Text
	var laySummaries []bbl.Text
	var keywords []string
	var conference bbl.Conference
	var contributors []bbl.WorkContributor

	var refresh string

	b := binder.New(r)

	b.Form().String("refresh", &refresh)

	if refresh == "" {
		b.Form().Vacuum()
	}

	b.Form().String("kind", &kind).
		String("sub_kind", &subKind).
		Each("identifiers", func(b *binder.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			if refresh != "" || code.Val != "" {
				identifiers = append(identifiers, code)
			}
			return true
		}).
		Each("titles", func(b *binder.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			if refresh != "" || text.Val != "" {
				titles = append(titles, text)
			}
			return true
		}).
		Each("abstracts", func(b *binder.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			if refresh != "" || text.Val != "" {
				abstracts = append(abstracts, text)
			}
			return true
		}).
		Each("lay_summaries", func(b *binder.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			if refresh != "" || text.Val != "" {
				laySummaries = append(laySummaries, text)
			}
			return true
		}).
		StringSlice("keywords", &keywords).
		String("conference.name", &conference.Name).
		String("conference.organizer", &conference.Organizer).
		String("conference.location", &conference.Location).
		Each("contributors", func(b *binder.Values) bool {
			var con bbl.WorkContributor
			b.String("attrs.name", &con.Attrs.Name)
			b.String("attrs.given_name", &con.Attrs.GivenName)
			b.String("attrs.middle_name", &con.Attrs.MiddleName)
			b.String("attrs.family_name", &con.Attrs.FamilyName)
			b.String("person_id", &con.PersonID)
			contributors = append(contributors, con)
			return true
		})

	// manipulate and validate form
	if refresh != "" {
		switch {
		case b.Form().Has("identifiers.add_at"):
			var at int
			b.Form().Int("identifiers.add_at", &at)
			identifiers = slices.Grow(identifiers, 1)
			identifiers = slices.Insert(identifiers, at, bbl.Code{})
		case b.Form().Has("identifiers.remove_at"):
			var at int
			b.Form().Int("identifiers.remove_at", &at)
			identifiers = slices.Delete(identifiers, at, at+1)
		case b.Form().Has("titles.add_at"):
			var at int
			b.Form().Int("titles.add_at", &at)
			titles = slices.Grow(titles, 1)
			titles = slices.Insert(titles, at, bbl.Text{})
		case b.Form().Has("titles.remove_at"):
			var at int
			b.Form().Int("titles.remove_at", &at)
			titles = slices.Delete(titles, at, at+1)
		case b.Form().Has("abstracts.add_at"):
			var at int
			var text bbl.Text
			b.Form().Int("abstracts.add_at", &at).
				String("abstracts.add.lang", &text.Lang).
				String("abstracts.add.val", &text.Val)
			abstracts = slices.Grow(abstracts, 1)
			abstracts = slices.Insert(abstracts, at, text)
		case b.Form().Has("abstracts.edit_at"):
			var at int
			var text bbl.Text
			b.Form().Int("abstracts.edit_at", &at)
			b.Form().
				String(fmt.Sprintf("abstracts[%d].edit.lang", at), &text.Lang).
				String(fmt.Sprintf("abstracts[%d].edit.val", at), &text.Val)
			abstracts[at] = text
		case b.Form().Has("abstracts.remove_at"):
			var at int
			b.Form().Int("abstracts.remove_at", &at)
			abstracts = slices.Delete(abstracts, at, at+1)
		}
	}

	if err := b.Err(); err != nil {
		return "", err
	}

	rec.Kind = kind
	rec.SubKind = subKind
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return "", err
	}

	rec.Identifiers = identifiers
	rec.Contributors = contributors
	rec.Attrs.Titles = titles
	rec.Attrs.Abstracts = abstracts
	rec.Attrs.LaySummaries = laySummaries
	rec.Attrs.Keywords = keywords
	rec.Attrs.Conference = conference

	return refresh, nil
}
