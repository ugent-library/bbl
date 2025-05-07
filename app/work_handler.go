package app

import (
	"encoding/json"
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

type WorkCtx struct {
	*AppCtx
	Work *bbl.Work
}

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

func (h *WorkHandler) BindWork(r *http.Request, c *AppCtx) (*WorkCtx, error) {
	work, err := h.repo.GetWork(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		return nil, err
	}
	return &WorkCtx{AppCtx: c, Work: work}, nil
}

func (h *WorkHandler) BindWorkState(r *http.Request, c *AppCtx) (*WorkCtx, error) {
	var recState string
	var rec bbl.Work
	if err := binder.New(r).Form().String("work.state", &recState).Err(); err != nil {
		return nil, err
	}
	if err := c.DecryptValue(recState, &rec); err != nil {
		return nil, err
	}
	if err := bbl.LoadWorkProfile(&rec); err != nil {
		return nil, err
	}
	if err := bindWorkForm(r, &rec); err != nil {
		return nil, err
	}
	return &WorkCtx{AppCtx: c, Work: &rec}, nil
}

func (h *WorkHandler) AddRoutes(router *mux.Router, appCtx *ctx.Ctx[*AppCtx]) {
	searchCtx := ctx.Derive(appCtx, BindSearch)
	workCtx := ctx.Derive(appCtx, h.BindWork)
	workStateCtx := ctx.Derive(appCtx, h.BindWorkState)

	router.Handle("/works", searchCtx.Bind(h.Search)).Methods("GET").Name("works")
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works/_change_kind", workStateCtx.Bind(h.ChangeKind)).Methods("POST").Name("work_change_kind")
	router.Handle("/works/_add_identifier", workStateCtx.Bind(h.AddIdentifier)).Methods("POST").Name("work_add_identifier")
	router.Handle("/works/_remove_identifier", workStateCtx.Bind(h.RemoveIdentifier)).Methods("POST").Name("work_remove_identifier")
	router.Handle("/works/_suggest_contributor", appCtx.Bind(h.SuggestContributor)).Methods("GET").Name("work_suggest_contributor")
	router.Handle("/works/_add_contributor", workStateCtx.Bind(h.AddContributor)).Methods("POST").Name("work_add_contributor")
	router.Handle("/works/_edit_contributor", workStateCtx.Bind(h.EditContributor)).Methods("POST").Name("work_edit_contributor")
	router.Handle("/works/_remove_contributor", workStateCtx.Bind(h.RemoveContributor)).Methods("POST").Name("work_remove_contributor")
	router.Handle("/works/_add_files", workStateCtx.Bind(h.AddFiles)).Methods("POST").Name("work_add_files")
	router.Handle("/works/_add_title", workStateCtx.Bind(h.AddTitle)).Methods("POST").Name("work_add_title")
	router.Handle("/works/_remove_title", workStateCtx.Bind(h.RemoveTitle)).Methods("POST").Name("work_remove_title")
	router.Handle("/works/_add_abstract", workStateCtx.Bind(h.AddAbstract)).Methods("POST").Name("work_add_abstract")
	router.Handle("/works/_edit_abstract", workStateCtx.Bind(h.EditAbstract)).Methods("POST").Name("work_edit_abstract")
	router.Handle("/works/_remove_abstract", workStateCtx.Bind(h.RemoveAbstract)).Methods("POST").Name("work_remove_abstract")
	router.Handle("/works/_add_lay_summary", workStateCtx.Bind(h.AddLaySummary)).Methods("POST").Name("work_add_lay_summary")
	router.Handle("/works/_edit_lay_summary", workStateCtx.Bind(h.EditLaySummary)).Methods("POST").Name("work_edit_lay_summary")
	router.Handle("/works/_remove_lay_summary", workStateCtx.Bind(h.RemoveLaySummary)).Methods("POST").Name("work_remove_lay_summary")
	router.Handle("/works", workStateCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	router.Handle("/works/{id}", workCtx.Bind(h.Show)).Methods("GET").Name("work")
	router.Handle("/works/{id}/edit", workCtx.Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{id}", workStateCtx.Bind(h.Update)).Methods("POST").Name("update_work")
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

	// TODO this is repeated in refreshForm
	if rec.Profile.Identifiers != nil {
		rec.Identifiers = []bbl.Code{{}}
	}
	if rec.Profile.Titles != nil {
		rec.Attrs.Titles = []bbl.Text{{}}
	}

	state, err := c.EncryptValue(rec)
	if err != nil {
		return err
	}

	return workviews.Edit(c.ViewCtx(), rec, state).Render(r.Context(), w)
}

func (h *WorkHandler) Create(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	vacuumWork(c.Work)

	c.Work.ID = bbl.NewID()

	rev := bbl.NewRev()
	rev.Add(&bbl.CreateWork{Work: c.Work})
	if err := h.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	// TODO this is clunky
	rec, err := h.repo.GetWork(r.Context(), c.Work.ID)
	if err != nil {
		return err
	}
	c.Work = rec

	htmx.PushURL(w, c.Route("edit_work", "id", rec.ID).String())

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	state, err := c.EncryptValue(c.Work)
	if err != nil {
		return err
	}

	return workviews.Edit(c.ViewCtx(), c.Work, state).Render(r.Context(), w)
}

func (h *WorkHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	vacuumWork(c.Work)

	rev := bbl.NewRev()
	rev.Add(&bbl.UpdateWork{Work: c.Work})
	if err := h.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	// TODO this is clunky
	rec, err := h.repo.GetWork(r.Context(), c.Work.ID)
	if err != nil {
		return err
	}
	c.Work = rec

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) ChangeKind(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	err := binder.New(r).Form().
		String("kind", &c.Work.Kind).
		String("subkind", &c.Work.Subkind).
		Err()
	if err != nil {
		return err
	}

	if err := bbl.LoadWorkProfile(c.Work); err != nil {
		return err
	}

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) AddIdentifier(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}
	c.Work.Identifiers = slices.Insert(slices.Grow(c.Work.Identifiers, 1), idx, bbl.Code{})

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) RemoveIdentifier(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Identifiers = slices.Delete(c.Work.Identifiers, idx, idx+1)
	if len(c.Work.Identifiers) == 0 {
		c.Work.Identifiers = []bbl.Code{{}}
	}

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) SuggestContributor(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	var query string
	var action string
	var idx int
	err := binder.New(r).Query().
		String("q", &query).
		String("action", &action).
		Int("idx", &idx).
		Err()
	if err != nil {
		return err
	}

	hits, err := h.index.People().Search(r.Context(), bbl.SearchOpts{Query: query, Size: 20})
	if err != nil {
		return err
	}

	return workviews.ContributorSuggestions(c.ViewCtx(), hits, action, idx).Render(r.Context(), w)
}

func (h *WorkHandler) AddContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var personID string
	err := binder.New(r).Form().
		Int("idx", &idx).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	person, err := h.repo.GetPerson(r.Context(), personID)
	if err != nil {
		return err
	}

	c.Work.Contributors = slices.Grow(c.Work.Contributors, 1)
	c.Work.Contributors = slices.Insert(c.Work.Contributors, idx, bbl.WorkContributor{PersonID: personID, Person: person})

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) EditContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var personID string
	err := binder.New(r).Form().
		Int("idx", &idx).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	person, err := h.repo.GetPerson(r.Context(), personID)
	if err != nil {
		return err
	}

	c.Work.Contributors[idx] = bbl.WorkContributor{PersonID: personID, Person: person}

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) RemoveContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Contributors = slices.Delete(c.Work.Contributors, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) AddFiles(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	for _, str := range binder.New(r).Form().GetAll("files") {
		var f bbl.WorkFile
		if err := json.Unmarshal([]byte(str), &f); err != nil {
			return err
		}
		c.Work.Files = append(c.Work.Files, f)
	}

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) AddTitle(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Attrs.Titles = slices.Insert(slices.Grow(c.Work.Attrs.Titles, 1), idx, bbl.Text{})

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) RemoveTitle(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Attrs.Titles = slices.Delete(c.Work.Attrs.Titles, idx, idx+1)
	if len(c.Work.Attrs.Titles) == 0 {
		c.Work.Attrs.Titles = []bbl.Text{{}}
	}

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) AddAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := binder.New(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Attrs.Abstracts = slices.Insert(slices.Grow(c.Work.Attrs.Abstracts, 1), idx, text)

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) EditAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := binder.New(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Attrs.Abstracts[idx] = text

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) RemoveAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Attrs.Abstracts = slices.Delete(c.Work.Attrs.Abstracts, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) AddLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := binder.New(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Attrs.LaySummaries = slices.Insert(slices.Grow(c.Work.Attrs.LaySummaries, 1), idx, text)

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) EditLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := binder.New(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Attrs.LaySummaries[idx] = text

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) RemoveLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Attrs.LaySummaries = slices.Delete(c.Work.Attrs.LaySummaries, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorkHandler) refreshForm(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	// TODO this is repeated in New
	if c.Work.Profile.Identifiers != nil && len(c.Work.Identifiers) == 0 {
		c.Work.Identifiers = []bbl.Code{{}}
	}
	if c.Work.Profile.Titles != nil && len(c.Work.Attrs.Titles) == 0 {
		c.Work.Attrs.Titles = []bbl.Text{{}}
	}

	state, err := c.EncryptValue(c.Work)
	if err != nil {
		return err
	}

	return workviews.RefreshForm(c.ViewCtx(), c.Work, state).Render(r.Context(), w)
}

func bindWorkForm(r *http.Request, rec *bbl.Work) error {
	// we only need to bind inline editable fields
	err := binder.New(r).Form().
		Each("work.identifiers", func(i int, b *binder.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Identifiers[i] = code
			return true
		}).
		Each("work.titles", func(i int, b *binder.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.Attrs.Titles[i] = text
			return true
		}).
		String("work.conference.name", &rec.Attrs.Conference.Name).
		String("work.conference.organizer", &rec.Attrs.Conference.Organizer).
		String("work.conference.location", &rec.Attrs.Conference.Location).
		Err()
	return err
}

func vacuumWork(rec *bbl.Work) {
	var identifiers []bbl.Code
	var titles []bbl.Text
	for _, code := range rec.Identifiers {
		if code.Val != "" {
			identifiers = append(identifiers, code)
		}
	}
	for _, text := range rec.Attrs.Titles {
		if text.Val != "" {
			titles = append(titles, text)
		}
	}
	rec.Identifiers = identifiers
	rec.Attrs.Titles = titles
}
