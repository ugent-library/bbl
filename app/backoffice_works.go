package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/urls"
	"github.com/ugent-library/bbl/app/views"
	workviews "github.com/ugent-library/bbl/app/views/backoffice/works"
	"github.com/ugent-library/bbl/bind"
	"github.com/ugent-library/bbl/can"
	"github.com/ugent-library/bbl/httperr"
	"github.com/ugent-library/bbl/workflows"
)

func (app *App) bindSearchWorksOpts(r *http.Request, c *appCtx, scope string) (*bbl.SearchOpts, error) {
	opts, err := bindSearchOpts(r, nil)
	if err != nil {
		return nil, err
	}

	switch scope {
	case "curator":
		if !can.Curate(c.User) {
			return nil, httperr.Forbidden
		}
	case "creator":
		opts.AddTermsFilter("creator", c.User.ID)
	case "contributor":
		personIDs, err := app.repo.GetPeopleIDsByIdentifiers(r.Context(), c.User.Identifiers)
		if err != nil {
			return nil, err
		}
		opts.AddTermsFilter("contributor", personIDs...)
	default:
		return nil, httperr.BadRequest
	}
	return opts, nil
}

func (app *App) backofficeWorks(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		if can.Curate(c.User) {
			scope = "curator"
		} else {
			scope = "contributor"
		}

	}

	opts, err := app.bindSearchWorksOpts(r, c, scope)
	if err != nil {
		return err
	}

	hits, err := app.index.Works().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return workviews.Search(c.viewCtx(), scope, hits).Render(r.Context(), w)
}

func (app *App) backofficeExportWorks(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	format := r.PathValue("format")

	scope := r.FormValue("scope")
	if scope == "" {
		if can.Curate(c.User) {
			scope = "curator"
		} else {
			scope = "contributor"
		}

	}

	opts, err := app.bindSearchWorksOpts(r, c, scope)
	if err != nil {
		return err
	}

	// TODO
	opts.From = 0
	opts.Cursor = ""
	opts.Size = 100
	opts.Facets = nil

	// TODO do something with ref
	_, err = app.exportWorksTask.RunNoWait(r.Context(), workflows.ExportWorksInput{
		UserID: c.User.ID,
		Opts:   opts,
		Format: format,
	})
	if err != nil {
		return err
	}

	return views.AddFlash(views.FlashArgs{
		Type:         views.FlashInfo,
		Title:        "Export started",
		Text:         "You will be notified when your export is ready.",
		DismissAfter: 5 * time.Second,
	}).Render(r.Context(), w)
}

func (app *App) backofficeAddWork(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return workviews.Add(c.viewCtx()).Render(r.Context(), w)
}

func (app *App) backofficeCreateWork(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	kind := r.FormValue("kind")

	rec := &bbl.Work{
		RecHeader: bbl.RecHeader{
			ID: bbl.NewID(),
		},
		Permissions: []bbl.Permission{{Kind: "edit", UserID: c.User.ID}}, // TODO autoadd in repo?
		Status:      bbl.DraftStatus,
		Kind:        kind,
	}
	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	rev := &bbl.Rev{UserID: c.User.ID}
	rev.Add(&bbl.CreateWork{Work: rec})
	if err := app.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	http.Redirect(w, r, urls.BackofficeEditWork(rec.ID), http.StatusSeeOther)

	return nil
}

func (app *App) backofficeEditWork(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := app.repo.GetWork(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	state, err := c.crypt.EncryptValue(rec)
	if err != nil {
		return err
	}

	return workviews.Edit(c.viewCtx(), rec, state).Render(r.Context(), w)
}

func (app *App) backofficeUpdateWork(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	work, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	vacuumWork(work)

	rev := &bbl.Rev{UserID: c.User.ID}
	rev.Add(&bbl.UpdateWork{Work: work, MatchVersion: true})
	if err := app.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	// TODO this is clunky
	rec, err := app.repo.GetWork(r.Context(), work.ID)
	if err != nil {
		return err
	}
	work = rec

	return app.refreshWorkForm(w, r, c, work)
}

func (app *App) backofficeWorkChangeKind(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	err = bind.Request(r).Form().
		String("kind", &rec.Kind).
		String("subkind", &rec.Subkind).
		Err()
	if err != nil {
		return err
	}

	if err := bbl.LoadWorkProfile(rec); err != nil {
		return err
	}

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkSuggestContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
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

	hits, err := app.index.People().Search(r.Context(), &bbl.SearchOpts{Query: query, Size: 20})
	if err != nil {
		return err
	}

	return workviews.ContributorSuggestions(c.viewCtx(), hits, action, idx).Render(r.Context(), w)
}

func (app *App) backofficeWorkAddContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	var idx int
	var creditRoles []string
	var personID string
	err = bind.Request(r).Form().
		Int("idx", &idx).
		StringSlice("credit_roles", &creditRoles).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	person, err := app.repo.GetPerson(r.Context(), personID)
	if err != nil {
		return err
	}

	rec.Contributors = slices.Grow(rec.Contributors, 1)
	rec.Contributors = slices.Insert(rec.Contributors, idx, bbl.WorkContributor{
		WorkContributorAttrs: bbl.WorkContributorAttrs{
			CreditRoles: creditRoles,
		},
		PersonID: personID,
		Person:   person,
	})

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkEditContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	var idx int
	var creditRoles []string
	var personID string
	err = bind.Request(r).Form().
		Int("idx", &idx).
		StringSlice("credit_roles", &creditRoles).
		String("person_id", &personID).
		Err()
	if err != nil {
		return err
	}

	person, err := app.repo.GetPerson(r.Context(), personID)
	if err != nil {
		return err
	}

	rec.Contributors[idx] = bbl.WorkContributor{
		WorkContributorAttrs: bbl.WorkContributorAttrs{
			CreditRoles: creditRoles,
		},
		PersonID: personID,
		Person:   person,
	}

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkRemoveContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	rec.Contributors = slices.Delete(rec.Contributors, idx, idx+1)

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkAddFiles(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	for _, str := range bind.Request(r).Form().GetAll("files") {
		var f bbl.WorkFile
		if err := json.Unmarshal([]byte(str), &f); err != nil {
			return err
		}
		rec.Files = append(rec.Files, f)
	}

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkRemoveFile(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := bindWorkState(r, c)
	if err != nil {
		return err
	}

	var idx int
	if err := bind.Request(r).Form().Int("idx", &idx).Err(); err != nil {
		return err
	}

	rec.Files = slices.Delete(rec.Files, idx, idx+1)

	return app.refreshWorkForm(w, r, c, rec)
}

func (app *App) backofficeWorkChanges(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := app.repo.GetWork(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	changes, err := app.repo.GetWorkChanges(r.Context(), rec.ID)
	if err != nil {
		return err
	}

	return workviews.Changes(c.viewCtx(), rec, changes).Render(r.Context(), w)
}

func (app *App) backofficeBatchEditWorks(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return workviews.BatchEdit(c.viewCtx(), workviews.BatchEditArgs{}).Render(r.Context(), w)
}

func (app *App) backofficeBatchUpdateWorks(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	args := workviews.BatchEditArgs{}
	args.Value = strings.ReplaceAll(strings.TrimSpace(r.FormValue("changes")), "\r\n", "\n")

	lines := strings.Split(args.Value, "\n")
	if len(lines) > 500 {
		args.Errors = []string{"no more than 500 changes can be processed at a time"}
		return workviews.BatchEdit(c.viewCtx(), args).Render(r.Context(), w)
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

	if err := app.repo.AddRev(r.Context(), rev); err != nil {
		args.Errors = append(args.Errors, fmt.Sprintf("could not process works: %s", err))
	}

	if len(args.Errors) == 0 {
		args.Done = len(lines)
		args.Value = ""
	}

	return workviews.BatchEdit(c.viewCtx(), args).Render(r.Context(), w)
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

func bindWorkState(r *http.Request, c *appCtx) (*bbl.Work, error) {
	var recState string
	var rec bbl.Work
	if err := bind.Request(r).Form().String("work.state", &recState).Err(); err != nil {
		return nil, err
	}
	if err := c.crypt.DecryptValue(recState, &rec); err != nil {
		return nil, err
	}
	if err := bbl.LoadWorkProfile(&rec); err != nil {
		return nil, err
	}
	if err := bindWorkForm(r, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

func bindWorkForm(r *http.Request, rec *bbl.Work) error {
	// TODO bind could use a Map function
	rec.Identifiers = nil
	rec.Classifications = nil
	rec.Titles = nil
	rec.Abstracts = nil
	rec.LaySummaries = nil
	err := bind.Request(r).Form().
		Each("work.identifiers", func(i int, b *bind.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Identifiers = append(rec.Identifiers, code)
			return true
		}).
		Each("work.classifications", func(i int, b *bind.Values) bool {
			var code bbl.Code
			b.String("scheme", &code.Scheme)
			b.String("val", &code.Val)
			rec.Classifications = append(rec.Classifications, code)
			return true
		}).
		Each("work.titles", func(i int, b *bind.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.Titles = append(rec.Titles, text)
			return true
		}).
		Each("work.abstracts", func(i int, b *bind.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.Abstracts = append(rec.Abstracts, text)
			return true
		}).
		Each("work.lay_summaries", func(i int, b *bind.Values) bool {
			var text bbl.Text
			b.String("lang", &text.Lang)
			b.String("val", &text.Val)
			rec.LaySummaries = append(rec.LaySummaries, text)
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

func (app *App) refreshWorkForm(w http.ResponseWriter, r *http.Request, c *appCtx, rec *bbl.Work) error {
	state, err := c.crypt.EncryptValue(rec)
	if err != nil {
		return err
	}

	return workviews.RefreshForm(c.viewCtx(), rec, state).Render(r.Context(), w)
}
