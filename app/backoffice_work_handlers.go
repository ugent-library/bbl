package app

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"slices"
	"strconv"
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

	// log search queries
	if hits.Opts.Query != "" && hits.Opts.Cursor == "" && hits.Opts.From == 0 && hits.Total > 0 {
		if err := app.repo.AddWorkSearch(r.Context(), hits.Opts.Query); err != nil {
			return err
		}
	}

	return workviews.Search(c.viewCtx(), scope, hits).Render(r.Context(), w)
}

func (app *App) backofficeWorksSuggest(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	q := r.URL.Query().Get("q")

	hits, err := app.index.WorkSearches().Search(r.Context(), q)
	if err != nil {
		return err
	}

	return workviews.Suggest(c.viewCtx(), hits).Render(r.Context(), w)
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

	return workviews.Edit(c.viewCtx(), rec).Render(r.Context(), w)
}

func (app *App) backofficeUpdateWork(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := app.repo.GetWork(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	if err := bindWorkForm(r, rec); err != nil {
		return err
	}

	vacuumWork(rec)

	rev := &bbl.Rev{UserID: c.User.ID}
	rev.Add(&bbl.UpdateWork{Work: rec, MatchVersion: true})
	if err := app.repo.AddRev(r.Context(), rev); err != nil {
		return err
	}

	// TODO this is clunky, there should be a convenience method for save and reload
	rec, err = app.repo.GetWork(r.Context(), rec.ID)
	if err != nil {
		return err
	}

	return workviews.RefreshForm(c.viewCtx(), rec).Render(r.Context(), w)
}

func (app *App) backofficeWorkChangeKind(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	rec, err := app.repo.GetWork(r.Context(), r.PathValue("id"))
	if err != nil {
		return err
	}

	if err := bindWorkForm(r, rec); err != nil {
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

	return workviews.RefreshForm(c.viewCtx(), rec).Render(r.Context(), w)
}

func (app *App) backofficeWorkAddContributorSuggest(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	query := r.URL.Query().Get("q")

	hits, err := app.index.People().Search(r.Context(), &bbl.SearchOpts{Query: query, Size: 20})
	if err != nil {
		return err
	}

	return workviews.AddContributorSuggest(c.viewCtx(), hits).Render(r.Context(), w)
}

// TODO should be backofficeWorkCreateContributor
func (app *App) backofficeWorkAddContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var cons []bbl.WorkContributor
	var personID string
	var name string
	var creditRoles []string

	err := bind.Request(r).Form().
		JSON("work.contributors", &cons).
		String("person_id", &personID).
		String("name", &name).
		StringSlice("credit_roles", &creditRoles).
		Err()
	if err != nil {
		return err
	}

	if personID != "" {
		for _, con := range cons {
			if con.PersonID == personID {
				return workviews.RefreshContributors(c.viewCtx(), cons).Render(r.Context(), w)
			}
		}

		rec, err := app.repo.GetPerson(r.Context(), personID)
		if err != nil {
			return err
		}

		cons = append(cons, bbl.WorkContributor{
			WorkContributorAttrs: bbl.WorkContributorAttrs{
				CreditRoles: creditRoles,
			},
			PersonID: personID,
			Person:   rec,
		})
	} else {
		cons = append(cons, bbl.WorkContributor{
			WorkContributorAttrs: bbl.WorkContributorAttrs{
				Name:        name,
				CreditRoles: creditRoles,
			},
		})
	}

	return workviews.RefreshContributors(c.viewCtx(), cons).Render(r.Context(), w)
}

func (app *App) backofficeWorkEditContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var cons []bbl.WorkContributor
	var idx int

	err := bind.Request(r).Form().
		JSON("work.contributors", &cons).
		Int("idx", &idx).
		Err()
	if err != nil {
		return err
	}

	con := cons[idx]

	return workviews.EditContributor(c.viewCtx(), con, idx).Render(r.Context(), w)
}

func (app *App) backofficeWorkEditContributorSuggest(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var query string
	var idx int

	err := bind.Request(r).Query().
		String("q", &query).
		Int("idx", &idx).
		Err()
	if err != nil {
		return err
	}

	hits, err := app.index.People().Search(r.Context(), &bbl.SearchOpts{Query: query, Size: 20})
	if err != nil {
		return err
	}

	return workviews.EditContributorSuggest(c.viewCtx(), hits, idx).Render(r.Context(), w)
}

func (app *App) backofficeWorkUpdateContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var cons []bbl.WorkContributor
	var personID string
	var name string
	var creditRoles []string

	idx, err := strconv.Atoi(r.PathValue("idx"))
	if err != nil {
		return err
	}

	err = bind.Request(r).Form().
		JSON("work.contributors", &cons).
		String("person_id", &personID).
		String("name", &name).
		StringSlice("credit_roles", &creditRoles).
		Err()
	if err != nil {
		return err
	}

	if personID != "" {
		rec, err := app.repo.GetPerson(r.Context(), personID)
		if err != nil {
			return err
		}

		cons[idx] = bbl.WorkContributor{
			WorkContributorAttrs: bbl.WorkContributorAttrs{
				CreditRoles: creditRoles,
			},
			PersonID: personID,
			Person:   rec,
		}
	} else {
		cons[idx] = bbl.WorkContributor{
			WorkContributorAttrs: bbl.WorkContributorAttrs{
				Name:        name,
				CreditRoles: creditRoles,
			},
		}
	}

	return workviews.RefreshContributors(c.viewCtx(), cons).Render(r.Context(), w)
}

func (app *App) backofficeWorkRemoveContributor(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var cons []bbl.WorkContributor
	var idx int

	err := bind.Request(r).Form().
		JSON("work.contributors", &cons).
		Int("idx", &idx).
		Err()
	if err != nil {
		return err
	}

	cons = slices.Delete(cons, idx, idx+1)

	return workviews.RefreshContributors(c.viewCtx(), cons).Render(r.Context(), w)
}

func (app *App) backofficeWorkAddFiles(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var files []bbl.WorkFile
	var newFiles []bbl.WorkFile

	err := bind.Request(r).Form().
		JSON("work.files", &files).
		JSON("files", &newFiles).
		Err()
	if err != nil {
		return err
	}

	files = append(files, newFiles...)

	return workviews.RefreshFiles(c.viewCtx(), files).Render(r.Context(), w)
}

func (app *App) backofficeWorkRemoveFile(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var files []bbl.WorkFile
	var idx int

	err := bind.Request(r).Form().
		JSON("work.files", &files).
		Int("idx", &idx).
		Err()
	if err != nil {
		return err
	}

	files = slices.Delete(files, idx, idx+1)

	return workviews.RefreshFiles(c.viewCtx(), files).Render(r.Context(), w)
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

// TODO make method of Work?
func vacuumWork(rec *bbl.Work) {
	var identifiers []bbl.Code
	var titles []bbl.Text
	var abstracts []bbl.Text
	var laySummaries []bbl.Text
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
	for _, text := range rec.Abstracts {
		if text.Val != "" {
			abstracts = append(abstracts, text)
		}
	}
	for _, text := range rec.LaySummaries {
		if text.Val != "" {
			laySummaries = append(laySummaries, text)
		}
	}
	rec.Identifiers = identifiers
	rec.Titles = titles
	rec.Abstracts = abstracts
	rec.LaySummaries = laySummaries
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
		JSON("work.contributors", &rec.Contributors).
		JSON("work.files", &rec.Files).
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
