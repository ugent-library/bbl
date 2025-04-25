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
	"github.com/ugent-library/htmx"
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
	searchCtx := ctx.Derive(appCtx, BindSearchCtx)
	router.Handle("/works", searchCtx.Bind(h.Search)).Methods("GET").Name("works")
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works", appCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	router.Handle("/works/suggest_contributors", appCtx.Bind(h.SuggestContributors)).Methods("GET").Name("work_suggest_contributors")
	router.Handle("/works/{id}", workCtx.Bind(h.Show)).Methods("GET").Name("work")
	router.Handle("/works/{id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{id}", workCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

func (h *WorkHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchCtx) error {
	c.SearchOpts.Facets = []string{"kind", "status"}

	hits, err := h.index.Works().Search(r.Context(), c.SearchOpts)
	if err != nil {
		return err
	}

	return workviews.Search(c.ViewCtx(), hits).Render(r.Context(), w)
}

func (h *WorkHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return workviews.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{Kind: bbl.WorkKinds[0]}
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	route := c.Route("create_work")

	return workviews.Edit(c.ViewCtx(), rec, route).Render(r.Context(), w)
}

func (h *WorkHandler) Create(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{}

	refresh, err := h.bindWorkForm(r, rec)
	if err != nil {
		return err
	}

	route := c.Route("create_work")

	if refresh != "" {
		return workviews.RefreshForm(c.ViewCtx(), rec, route).Render(r.Context(), w)
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

	route = c.Route("update_work", "id", rec.ID)

	htmx.PushURL(w, c.Route("edit_work", "id", rec.ID).String())

	return workviews.RefreshForm(c.ViewCtx(), rec, route).Render(r.Context(), w)
}

func (h *WorkHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	route := c.Route("update_work", "id", c.Work.ID)

	return workviews.Edit(c.ViewCtx(), c.Work, route).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	refresh, err := h.bindWorkForm(r, c.Work)
	if err != nil {
		return err
	}

	route := c.Route("update_work", "id", c.Work.ID)

	if refresh != "" {
		return workviews.RefreshForm(c.ViewCtx(), c.Work, route).Render(r.Context(), w)
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

	return workviews.RefreshForm(c.ViewCtx(), work, route).Render(r.Context(), w)
}

func (h *WorkHandler) SuggestContributors(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var query string
	var id string
	var addAt int = -1
	var editAt int = -1
	if err := binder.New(r).Query().
		String("q", &query).
		String("id", &id).
		Int("add_at", &addAt).
		Int("edit_at", &editAt).
		Err(); err != nil {
		return err
	}
	hits, err := h.index.People().Search(r.Context(), bbl.SearchOpts{Query: query, Size: 20})
	if err != nil {
		return err
	}
	return workviews.ContributorSuggestions(c.ViewCtx(), hits, id, addAt, editAt).Render(r.Context(), w)
}

func (h *WorkHandler) bindWorkForm(r *http.Request, rec *bbl.Work) (string, error) {
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
		String("subkind", &subKind).
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
		case b.Form().Has("lay_summaries.add_at"):
			var at int
			var text bbl.Text
			b.Form().Int("lay_summaries.add_at", &at).
				String("lay_summaries.add.lang", &text.Lang).
				String("lay_summaries.add.val", &text.Val)
			laySummaries = slices.Grow(laySummaries, 1)
			laySummaries = slices.Insert(laySummaries, at, text)
		case b.Form().Has("lay_summaries.edit_at"):
			var at int
			var text bbl.Text
			b.Form().Int("lay_summaries.edit_at", &at)
			b.Form().
				String(fmt.Sprintf("lay_summaries[%d].edit.lang", at), &text.Lang).
				String(fmt.Sprintf("lay_summaries[%d].edit.val", at), &text.Val)
			laySummaries[at] = text
		case b.Form().Has("lay_summaries.remove_at"):
			var at int
			b.Form().Int("lay_summaries.remove_at", &at)
			laySummaries = slices.Delete(laySummaries, at, at+1)
		case b.Form().Has("contributors.add_at"):
			var at int
			var personID string
			b.Form().Int("contributors.add_at", &at).
				String("person_id", &personID)
			contributors = slices.Grow(contributors, 1)
			contributors = slices.Insert(contributors, at, bbl.WorkContributor{PersonID: personID})
		case b.Form().Has("contributors.edit_at"):
			var at int
			var personID string
			b.Form().Int("contributors.edit_at", &at).
				String("person_id", &personID)
			contributors[at] = bbl.WorkContributor{PersonID: personID}
		case b.Form().Has("contributors.remove_at"):
			var at int
			b.Form().Int("contributors.remove_at", &at)
			contributors = slices.Delete(contributors, at, at+1)
		}
	}

	if err := b.Err(); err != nil {
		return "", err
	}

	for i, con := range contributors {
		if con.PersonID != "" {
			person, err := h.repo.GetPerson(r.Context(), con.PersonID)
			if err != nil {
				return "", err
			}
			contributors[i].Person = person
		}
	}

	rec.Kind = kind
	rec.Subkind = subKind
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
