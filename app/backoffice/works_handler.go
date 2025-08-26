package backoffice

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/ctx"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/app/views/backoffice/works"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/can"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/htmx"
	"github.com/ugent-library/httperror"
)

type WorkCtx struct {
	*ctx.Ctx
	Work *bbl.Work
}

func RequireCanViewWork(next bind.Handler[*WorkCtx]) bind.Handler[*WorkCtx] {
	return bind.HandlerFunc[*WorkCtx](func(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
		if can.ViewWork(c.User, c.Work) {
			return next.ServeHTTP(w, r, c)
		}
		return httperror.Forbidden
	})
}

func RequireCanEditWork(next bind.Handler[*WorkCtx]) bind.Handler[*WorkCtx] {
	return bind.HandlerFunc[*WorkCtx](func(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
		if can.EditWork(c.User, c.Work) {
			return next.ServeHTTP(w, r, c)
		}
		return httperror.Forbidden
	})
}

type WorksHandler struct {
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewWorksHandler(repo *pgxrepo.Repo, index bbl.Index) *WorksHandler {
	return &WorksHandler{
		repo:  repo,
		index: index,
	}
}

func (h *WorksHandler) WorkBinder(r *http.Request, c *ctx.Ctx) (*WorkCtx, error) {
	work, err := h.repo.GetWork(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		return nil, err
	}
	return &WorkCtx{Ctx: c, Work: work}, nil
}

func (h *WorksHandler) WorkStateBinder(r *http.Request, c *ctx.Ctx) (*WorkCtx, error) {
	var recState string
	var rec bbl.Work
	if err := bind.Request(r).Form().String("work.state", &recState).Err(); err != nil {
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
	return &WorkCtx{Ctx: c, Work: &rec}, nil
}

func (h *WorksHandler) AddRoutes(r *mux.Router, b *bind.HandlerBinder[*ctx.Ctx]) {
	searchBinder := bind.Derive(b, SearchBinder)
	workBinder := bind.Derive(b, h.WorkBinder)
	workStateBinder := bind.Derive(b, h.WorkStateBinder)

	r.Handle("/works", searchBinder.BindFunc(h.Search)).Methods("GET").Name("works")
	r.Handle("/works/export", searchBinder.BindFunc(h.Export)).Methods("POST").Name("export_works")
	r.Handle("/works/new", b.BindFunc(h.New)).Methods("GET").Name("new_work")
	r.Handle("/works/_change_kind", workStateBinder.BindFunc(h.ChangeKind)).Methods("POST").Name("work_change_kind")
	r.Handle("/works/_add_identifier", workStateBinder.BindFunc(h.AddIdentifier)).Methods("POST").Name("work_add_identifier")
	r.Handle("/works/_remove_identifier", workStateBinder.BindFunc(h.RemoveIdentifier)).Methods("POST").Name("work_remove_identifier")
	r.Handle("/works/_suggest_contributor", b.BindFunc(h.SuggestContributor)).Methods("GET").Name("work_suggest_contributor")
	r.Handle("/works/_add_contributor", workStateBinder.BindFunc(h.AddContributor)).Methods("POST").Name("work_add_contributor")
	r.Handle("/works/_edit_contributor", workStateBinder.BindFunc(h.EditContributor)).Methods("POST").Name("work_edit_contributor")
	r.Handle("/works/_remove_contributor", workStateBinder.BindFunc(h.RemoveContributor)).Methods("POST").Name("work_remove_contributor")
	r.Handle("/works/_add_files", workStateBinder.BindFunc(h.AddFiles)).Methods("POST").Name("work_add_files")
	r.Handle("/works/_remove_file", workStateBinder.BindFunc(h.RemoveFile)).Methods("POST").Name("work_remove_file")
	r.Handle("/works/_add_title", workStateBinder.BindFunc(h.AddTitle)).Methods("POST").Name("work_add_title")
	r.Handle("/works/_remove_title", workStateBinder.BindFunc(h.RemoveTitle)).Methods("POST").Name("work_remove_title")
	r.Handle("/works/_add_abstract", workStateBinder.BindFunc(h.AddAbstract)).Methods("POST").Name("work_add_abstract")
	r.Handle("/works/_edit_abstract", workStateBinder.BindFunc(h.EditAbstract)).Methods("POST").Name("work_edit_abstract")
	r.Handle("/works/_remove_abstract", workStateBinder.BindFunc(h.RemoveAbstract)).Methods("POST").Name("work_remove_abstract")
	r.Handle("/works/_add_lay_summary", workStateBinder.BindFunc(h.AddLaySummary)).Methods("POST").Name("work_add_lay_summary")
	r.Handle("/works/_edit_lay_summary", workStateBinder.BindFunc(h.EditLaySummary)).Methods("POST").Name("work_edit_lay_summary")
	r.Handle("/works/_remove_lay_summary", workStateBinder.BindFunc(h.RemoveLaySummary)).Methods("POST").Name("work_remove_lay_summary")
	r.Handle("/works", workStateBinder.BindFunc(h.Create)).Methods("POST").Name("create_work")
	r.Handle("/works/batch/edit", b.BindFunc(h.BatchEdit)).Methods("GET").Name("batch_edit_works")
	r.Handle("/works/batch", b.BindFunc(h.BatchUpdate)).Methods("POST").Name("batch_update_works")
	r.Handle("/works/{id}", workBinder.With(RequireCanViewWork).BindFunc(h.Show)).Methods("GET").Name("work")
	r.Handle("/works/{id}/_changes", workBinder.BindFunc(h.Changes)).Methods("GET").Name("work_changes")
	r.Handle("/works/{id}/edit", workBinder.With(RequireCanEditWork).BindFunc(h.Edit)).Methods("GET").Name("edit_work")
	r.Handle("/works/{id}", workStateBinder.BindFunc(h.Update)).Methods("POST").Name("update_work")
}

func (h *WorksHandler) setSearchScope(ctx context.Context, c *SearchCtx) error {
	switch c.Scope {
	case "curator":
		if !can.Curate(c.User) {
			return httperror.Forbidden
		}
	case "creator":
		c.Opts.AddFilters(bbl.Terms("creator", c.User.ID))
	case "contributor":
		personIDs, err := h.repo.GetPeopleIDsByIdentifiers(ctx, c.User.Identifiers)
		if err != nil {
			return err
		}
		c.Opts.AddFilters(bbl.Terms("contributor", personIDs...))
	default:
		return httperror.BadRequest
	}
	return nil
}

func (h *WorksHandler) Search(w http.ResponseWriter, r *http.Request, c *SearchCtx) error {
	if err := h.setSearchScope(r.Context(), c); err != nil {
		return err
	}

	hits, err := h.index.Works().Search(r.Context(), c.Opts)
	if err != nil {
		return err
	}

	return works.Search(c.ViewCtx(), c.Scope, hits).Render(r.Context(), w)
}

func (h *WorksHandler) Export(w http.ResponseWriter, r *http.Request, c *SearchCtx) error {
	if err := h.setSearchScope(r.Context(), c); err != nil {
		return err
	}

	// TODO
	c.Opts.From = 0
	c.Opts.Cursor = ""
	c.Opts.Size = 100
	c.Opts.Facets = nil
	format := r.FormValue("format")

	_, err := h.repo.AddJob(r.Context(), jobs.ExportWorks{UserID: c.User.ID, Opts: c.Opts, Format: format})
	if err != nil {
		return err
	}

	return c.Hub.Render(r.Context(), "users."+c.User.ID, "flash", views.Flash(views.FlashArgs{
		Type:         views.FlashInfo,
		Title:        "Export started",
		Text:         "You will be notified when your export is ready.",
		DismissAfter: 5 * time.Second,
	}))
}

func (h *WorksHandler) Show(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	return works.Show(c.ViewCtx(), c.Work).Render(r.Context(), w)
}

func (h *WorksHandler) New(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
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
		rec.Titles = []bbl.Text{{}}
	}

	state, err := c.EncryptValue(rec)
	if err != nil {
		return err
	}

	return works.Edit(c.ViewCtx(), rec, state).Render(r.Context(), w)
}

func (h *WorksHandler) Create(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
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

func (h *WorksHandler) Edit(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	state, err := c.EncryptValue(c.Work)
	if err != nil {
		return err
	}

	return works.Edit(c.ViewCtx(), c.Work, state).Render(r.Context(), w)
}

func (h *WorksHandler) Update(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
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

func (h *WorksHandler) ChangeKind(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	err := bind.Request(r).Form().
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

func (h *WorksHandler) AddIdentifier(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}
	c.Work.Identifiers = slices.Insert(slices.Grow(c.Work.Identifiers, 1), idx, bbl.Code{})

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveIdentifier(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Identifiers = slices.Delete(c.Work.Identifiers, idx, idx+1)
	if len(c.Work.Identifiers) == 0 {
		c.Work.Identifiers = []bbl.Code{{}}
	}

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) SuggestContributor(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	var query string
	var action string
	var idx int
	err := bind.Request(r).Query().
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

	return works.ContributorSuggestions(c.ViewCtx(), hits, action, idx).Render(r.Context(), w)
}

func (h *WorksHandler) AddContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var creditRoles []string
	var personID string
	err := bind.Request(r).Form().
		Int("idx", &idx).
		StringSlice("credit_roles", &creditRoles).
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
	c.Work.Contributors = slices.Insert(c.Work.Contributors, idx, bbl.WorkContributor{
		Attrs: bbl.WorkContributorAttrs{
			CreditRoles: creditRoles,
		},
		PersonID: personID,
		Person:   person,
	})

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) EditContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var creditRoles []string
	var personID string
	err := bind.Request(r).Form().
		Int("idx", &idx).
		StringSlice("credit_roles", &creditRoles).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	person, err := h.repo.GetPerson(r.Context(), personID)
	if err != nil {
		return err
	}

	c.Work.Contributors[idx] = bbl.WorkContributor{
		Attrs: bbl.WorkContributorAttrs{
			CreditRoles: creditRoles,
		},
		PersonID: personID,
		Person:   person,
	}

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveContributor(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Contributors = slices.Delete(c.Work.Contributors, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) AddFiles(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	for _, str := range bind.Request(r).Form().GetAll("files") {
		var f bbl.WorkFile
		if err := json.Unmarshal([]byte(str), &f); err != nil {
			return err
		}
		c.Work.Files = append(c.Work.Files, f)
	}

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveFile(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Files = slices.Delete(c.Work.Files, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) AddTitle(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Titles = slices.Insert(slices.Grow(c.Work.Titles, 1), idx, bbl.Text{})

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveTitle(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Titles = slices.Delete(c.Work.Titles, idx, idx+1)
	if len(c.Work.Titles) == 0 {
		c.Work.Titles = []bbl.Text{{}}
	}

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) AddAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := bind.Request(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Abstracts = slices.Insert(slices.Grow(c.Work.Abstracts, 1), idx, text)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) EditAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := bind.Request(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.Abstracts[idx] = text

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveAbstract(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.Abstracts = slices.Delete(c.Work.Abstracts, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) AddLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := bind.Request(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.LaySummaries = slices.Insert(slices.Grow(c.Work.LaySummaries, 1), idx, text)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) EditLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	var text bbl.Text
	if err := bind.Request(r).Form().Int("idx", &idx).String("lang", &text.Lang).String("val", &text.Val).Err(); err != nil {
		return err
	}

	c.Work.LaySummaries[idx] = text

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) RemoveLaySummary(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	c.Work.LaySummaries = slices.Delete(c.Work.LaySummaries, idx, idx+1)

	return h.refreshForm(w, r, c)
}

func (h *WorksHandler) refreshForm(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	// TODO this is repeated in New
	if c.Work.Profile.Identifiers != nil && len(c.Work.Identifiers) == 0 {
		c.Work.Identifiers = []bbl.Code{{}}
	}
	if c.Work.Profile.Titles != nil && len(c.Work.Titles) == 0 {
		c.Work.Titles = []bbl.Text{{}}
	}

	state, err := c.EncryptValue(c.Work)
	if err != nil {
		return err
	}

	return works.RefreshForm(c.ViewCtx(), c.Work, state).Render(r.Context(), w)
}

func (h *WorksHandler) Changes(w http.ResponseWriter, r *http.Request, c *WorkCtx) error {
	changes, err := h.repo.GetWorkChanges(r.Context(), c.Work.ID)
	if err != nil {
		return err
	}
	return works.Changes(c.ViewCtx(), c.Work, changes).Render(r.Context(), w)
}

func (h *WorksHandler) BatchEdit(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	return works.BatchEdit(c.ViewCtx(), works.BatchEditArgs{}).Render(r.Context(), w)
}

func (h *WorksHandler) BatchUpdate(w http.ResponseWriter, r *http.Request, c *ctx.Ctx) error {
	args := works.BatchEditArgs{}
	args.Value = strings.ReplaceAll(strings.TrimSpace(r.FormValue("changes")), "\r\n", "\n")

	lines := strings.Split(args.Value, "\n")
	if len(lines) > 500 {
		args.Errors = []string{"no more than 500 changes can be processed at a time"}
		return works.BatchEdit(c.ViewCtx(), args).Render(r.Context(), w)
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

		initChange, ok := bbl.WorkChangers[changeName]
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

	return works.BatchEdit(c.ViewCtx(), args).Render(r.Context(), w)
}

func vacuumWork(rec *bbl.Work) {
	var identifiers []bbl.Code
	var titles []bbl.Text
	for _, code := range rec.Identifiers {
		if code.Val != "" {
			identifiers = append(identifiers, code)
		}
	}
	for _, text := range rec.Titles {
		if text.Val != "" {
			titles = append(titles, text)
		}
	}
	rec.Identifiers = identifiers
	rec.Titles = titles
}

func bindWorkForm(r *http.Request, rec *bbl.Work) error {
	// we only need to bind inline editable fields
	err := bind.Request(r).Form().
		Each("work.identifiers", func(i int, b *bind.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Identifiers[i] = code
			return true
		}).
		Each("work.classifications", func(i int, b *bind.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Classifications[i] = code
			return true
		}).
		Each("work.titles", func(i int, b *bind.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.Titles[i] = text
			return true
		}).
		StringSlice("work.keywords", &rec.Keywords).
		String("work.conference.name", &rec.Conference.Name).
		String("work.conference.organizer", &rec.Conference.Organizer).
		String("work.conference.location", &rec.Conference.Location).
		String("work.article_number", &rec.ArticleNumber).
		String("work.report_number", &rec.ReportNumber).
		String("work.volume", &rec.Volume).
		String("work.issue", &rec.Issue).
		String("work.issue_title", &rec.IssueTitle).
		String("work.edition", &rec.Edition).
		String("work.total_pages", &rec.TotalPages).
		String("work.pages.start", &rec.Pages.Start).
		String("work.pages.end", &rec.Pages.End).
		String("work.place_of_publication", &rec.PlaceOfPublication).
		String("work.publisher", &rec.Publisher).
		String("work.publication_year", &rec.PublicationYear).
		String("work.journal_title", &rec.JournalTitle).
		String("work.journal_abbreviation", &rec.JournalAbbreviation).
		String("work.book_title", &rec.BookTitle).
		String("work.series_title", &rec.SeriesTitle).
		Err()
	return err
}
