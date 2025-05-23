package app

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	workviews "github.com/ugent-library/bbl/app/views/works"
	"github.com/ugent-library/bbl/binder"
	"github.com/ugent-library/bbl/can"
	"github.com/ugent-library/bbl/ctx"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/htmx"
	"github.com/ugent-library/httperror"
)

type WorkCtx struct {
	*AppCtx
	Work *bbl.Work
}

func RequireCanViewWork(w http.ResponseWriter, r *http.Request, c *WorkCtx) (*http.Request, error) {
	if !can.ViewWork(c.User, c.Work) {
		return nil, httperror.Forbidden
	}
	return r, nil
}

func RequireCanEditWork(w http.ResponseWriter, r *http.Request, c *WorkCtx) (*http.Request, error) {
	if !can.EditWork(c.User, c.Work) {
		return nil, httperror.Forbidden
	}
	return r, nil
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
	router.Handle("/works/export", searchCtx.Bind(h.Export)).Methods("POST").Name("export_works")
	router.Handle("/works/new", appCtx.Bind(h.New)).Methods("GET").Name("new_work")
	router.Handle("/works/_change_kind", workStateCtx.Bind(h.ChangeKind)).Methods("POST").Name("work_change_kind")
	router.Handle("/works/_add_identifier", workStateCtx.Bind(h.AddIdentifier)).Methods("POST").Name("work_add_identifier")
	router.Handle("/works/_remove_identifier", workStateCtx.Bind(h.RemoveIdentifier)).Methods("POST").Name("work_remove_identifier")
	router.Handle("/works/_suggest_contributor", appCtx.Bind(h.SuggestContributor)).Methods("GET").Name("work_suggest_contributor")
	router.Handle("/works/_add_contributor", workStateCtx.Bind(h.AddContributor)).Methods("POST").Name("work_add_contributor")
	router.Handle("/works/_edit_contributor", workStateCtx.Bind(h.EditContributor)).Methods("POST").Name("work_edit_contributor")
	router.Handle("/works/_remove_contributor", workStateCtx.Bind(h.RemoveContributor)).Methods("POST").Name("work_remove_contributor")
	router.Handle("/works/_add_files", workStateCtx.Bind(h.AddFiles)).Methods("POST").Name("work_add_files")
	router.Handle("/works/_remove_file", workStateCtx.Bind(h.RemoveFile)).Methods("POST").Name("work_remove_file")
	router.Handle("/works/_add_title", workStateCtx.Bind(h.AddTitle)).Methods("POST").Name("work_add_title")
	router.Handle("/works/_remove_title", workStateCtx.Bind(h.RemoveTitle)).Methods("POST").Name("work_remove_title")
	router.Handle("/works/_add_abstract", workStateCtx.Bind(h.AddAbstract)).Methods("POST").Name("work_add_abstract")
	router.Handle("/works/_edit_abstract", workStateCtx.Bind(h.EditAbstract)).Methods("POST").Name("work_edit_abstract")
	router.Handle("/works/_remove_abstract", workStateCtx.Bind(h.RemoveAbstract)).Methods("POST").Name("work_remove_abstract")
	router.Handle("/works/_add_lay_summary", workStateCtx.Bind(h.AddLaySummary)).Methods("POST").Name("work_add_lay_summary")
	router.Handle("/works/_edit_lay_summary", workStateCtx.Bind(h.EditLaySummary)).Methods("POST").Name("work_edit_lay_summary")
	router.Handle("/works/_remove_lay_summary", workStateCtx.Bind(h.RemoveLaySummary)).Methods("POST").Name("work_remove_lay_summary")
	router.Handle("/works", workStateCtx.Bind(h.Create)).Methods("POST").Name("create_work")
	router.Handle("/works/batch/edit", appCtx.Bind(h.BatchEdit)).Methods("GET").Name("batch_edit_works")
	router.Handle("/works/batch", appCtx.Bind(h.BatchUpdate)).Methods("POST").Name("batch_update_works")
	router.Handle("/works/{id}", workCtx.With(RequireCanViewWork).Bind(h.Show)).Methods("GET").Name("work")
	router.Handle("/works/{id}/edit", workCtx.With(RequireCanEditWork).Bind(h.Edit)).Methods("GET").Name("edit_work")
	router.Handle("/works/{id}", workStateCtx.Bind(h.Update)).Methods("POST").Name("update_work")
}

func (h *WorkHandler) setSearchScope(ctx context.Context, c *SearchCtx) error {
	switch c.Scope {
	case "created":
		c.Opts.AddFilters(bbl.Terms("created", c.User.ID))
	case "contributed":
		personIDs, err := h.repo.GetPeopleIDsByIdentifiers(ctx, c.User.Identifiers)
		if err != nil {
			return err
		}
		c.Opts.AddFilters(bbl.Terms("contributed", personIDs...))
	case "":
		personIDs, err := h.repo.GetPeopleIDsByIdentifiers(ctx, c.User.Identifiers)
		if err != nil {
			return err
		}
		c.Opts.AddFilters(bbl.Or(
			bbl.Terms("created", c.User.ID),
			bbl.Terms("contributed", personIDs...),
		))
	default:
		return httperror.BadRequest
	}
	return nil
}

func (h *WorkHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchCtx) error {
	if err := h.setSearchScope(r.Context(), c); err != nil {
		return err
	}

	hits, err := h.index.Works().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return workviews.Search(c.ViewCtx(), c.Scope, hits).Render(r.Context(), w)
}

func (h *WorkHandler) Export(w http.ResponseWriter, r *http.Request, c *SearchCtx) error {
	if err := h.setSearchScope(r.Context(), c); err != nil {
		return err
	}

	// TODO
	c.Opts.From = 0
	c.Opts.Cursor = ""
	c.Opts.Size = 100
	c.Opts.Facets = nil
	format := r.FormValue("format")

	// TODO do something with jobID
	_, err := h.repo.AddJob(r.Context(), jobs.ExportWorks{Opts: c.Opts, Format: format})
	if err != nil {
		return err
	}

	return nil
}

func (h *WorkHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return workviews.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorkHandler) New(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	rec := &bbl.Work{
		Permissions: []bbl.Permission{{Kind: "edit", UserID: c.User.ID}}, // TODO autoadd in repo?
		Kind:        bbl.WorkKinds[0],
	}

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

	rev := &bbl.Rev{UserID: c.User.ID}
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

	rev := &bbl.Rev{UserID: c.User.ID}
	rev.Add(&bbl.UpdateWork{Work: c.Work, MatchVersion: true})
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

	hits, err := h.index.People().Search(r.Context(), &bbl.SearchOpts{Query: query, Size: 20})
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

func (h *WorkHandler) RemoveFile(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := binder.New(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Files = slices.Delete(c.Work.Files, idx, idx+1)

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

func (h *WorkHandler) BatchEdit(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	return workviews.BatchEdit(c.ViewCtx(), workviews.BatchEditArgs{}).Render(r.Context(), w)
}

func (h *WorkHandler) BatchUpdate(w http.ResponseWriter, r *http.Request, c *AppCtx) error {
	args := workviews.BatchEditArgs{}
	args.Value = strings.ReplaceAll(strings.TrimSpace(r.FormValue("changes")), "\r\n", "\n")

	lines := strings.Split(args.Value, "\n")
	if len(lines) > 500 {
		args.Errors = []string{"no more than 500 changes can be processed at a time"}
		return workviews.BatchEdit(c.ViewCtx(), args).Render(r.Context(), w)
	}

	var actions []*bbl.ChangeWork

LINES:
	for lineIndex, line := range lines {
		line = strings.TrimSpace(line)

		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		reader := csv.NewReader(strings.NewReader(line))
		reader.TrimLeadingSpace = true
		rec, err := reader.Read()

		if err != nil {
			args.Errors = append(args.Errors, fmt.Sprintf("error parsing line %d", lineIndex+1))
			continue
		}

		if len(rec) < 2 {
			args.Errors = append(args.Errors, fmt.Sprintf("error parsing line %d", lineIndex+1))
			continue
		}

		id := strings.TrimSpace(rec[0])
		changeName := strings.TrimSpace(rec[1])
		changeArgs := rec[2:]
		for i, arg := range changeArgs {
			changeArgs[i] = strings.TrimSpace(arg)
			if changeArgs[i] == "" {
				args.Errors = append(args.Errors, fmt.Sprintf("argument %d is empty at line %d", i+1, lineIndex+1))
				continue LINES
			}
		}

		if id == "" {
			args.Errors = append(args.Errors, fmt.Sprintf("empty id at line %d", lineIndex+1))
			continue
		}

		if len(actions) == 0 || actions[len(actions)-1].WorkID != id {
			actions = append(actions, &bbl.ChangeWork{WorkID: id})
		}

		action := actions[len(actions)-1]

		initChange, ok := bbl.WorkChanges[changeName]
		if !ok {
			args.Errors = append(args.Errors, fmt.Sprintf("unknown change %s at line %d", changeName, lineIndex+1))
			continue
		}

		change := initChange()
		if err := change.UnmarshalArgs(changeArgs); err != nil {
			args.Errors = append(args.Errors, fmt.Sprintf("invalid arguments for change %s at line %d", changeName, lineIndex+1))
			continue
		}

		action.Changes = append(action.Changes, change)
	}

	rev := &bbl.Rev{UserID: c.User.ID}
	for _, a := range actions {
		rev.Add(a)
	}

	if err := h.repo.AddRev(r.Context(), rev); err != nil {
		args.Errors = append(args.Errors, fmt.Sprintf("could not process works: %s", err))
	}

	if len(args.Errors) == 0 {
		args.Done = len(lines)
		args.Value = ""
	}

	return workviews.BatchEdit(c.ViewCtx(), args).Render(r.Context(), w)
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
		Each("work.classifications", func(i int, b *binder.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Attrs.Classifications[i] = code
			return true
		}).
		Each("work.titles", func(i int, b *binder.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.Attrs.Titles[i] = text
			return true
		}).
		StringSlice("work.keywords", &rec.Attrs.Keywords).
		String("work.conference.name", &rec.Attrs.Conference.Name).
		String("work.conference.organizer", &rec.Attrs.Conference.Organizer).
		String("work.conference.location", &rec.Attrs.Conference.Location).
		String("work.article_number", &rec.Attrs.ArticleNumber).
		String("work.report_number", &rec.Attrs.ReportNumber).
		String("work.volume", &rec.Attrs.Volume).
		String("work.issue", &rec.Attrs.Issue).
		String("work.issue_title", &rec.Attrs.IssueTitle).
		String("work.edition", &rec.Attrs.Edition).
		String("work.total_pages", &rec.Attrs.TotalPages).
		String("work.pages.start", &rec.Attrs.Pages.Start).
		String("work.pages.end", &rec.Attrs.Pages.End).
		String("work.place_of_publication", &rec.Attrs.PlaceOfPublication).
		String("work.publisher", &rec.Attrs.Publisher).
		Err()
	return err
}
