package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
)

func (app *App) backofficeEditWork(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	work, err := app.getWork(r, "public", "private")
	if err != nil {
		return err
	}
	defs := app.services.Repo.Profiles.FieldDefs("work", work.Kind)
	if defs == nil {
		return fmt.Errorf("no profile for kind %q", work.Kind)
	}
	return views.BackofficeEditWork(c.ViewCtx, work, defs, nil).Render(r.Context(), w)
}

func (app *App) backofficeUpdateWork(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	work, err := app.getWork(r, "public", "private")
	if err != nil {
		return err
	}
	defs := app.services.Repo.Profiles.FieldDefs("work", work.Kind)
	if defs == nil {
		return fmt.Errorf("no profile for kind %q", work.Kind)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	updates := buildWorkUpdates(r, defs, work)

	if len(updates) > 0 {
		_, err = app.services.UpdateAndIndex(r.Context(), c.User, updates...)
		if err != nil {
			return fmt.Errorf("backofficeUpdateWork: %w", err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/backoffice/works/%s", work.ID), http.StatusSeeOther)
	return nil
}

// buildWorkUpdates builds Set/Unset updates from the form for all profile fields.
func buildWorkUpdates(r *http.Request, defs []bbl.FieldDef, work *bbl.Work) []any {
	var updates []any
	for _, f := range defs {
		switch f.Type {
		case "string":
			val := strings.TrimSpace(r.FormValue(f.Name))
			if val != "" {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: val})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "title":
			langs := r.Form[f.Name+".lang"]
			vals := r.Form[f.Name+".val"]
			var titles []bbl.Title
			for i := range min(len(langs), len(vals)) {
				lang := strings.TrimSpace(langs[i])
				val := strings.TrimSpace(vals[i])
				if lang != "" || val != "" {
					titles = append(titles, bbl.Title{Lang: lang, Val: val})
				}
			}
			if len(titles) > 0 {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: titles})
			}
		case "text":
			langs := r.Form[f.Name+".lang"]
			vals := r.Form[f.Name+".val"]
			var texts []bbl.Text
			for i := range min(len(langs), len(vals)) {
				lang := strings.TrimSpace(langs[i])
				val := strings.TrimSpace(vals[i])
				if lang != "" || val != "" {
					texts = append(texts, bbl.Text{Lang: lang, Val: val})
				}
			}
			if len(texts) > 0 {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: texts})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "keyword":
			var keywords []bbl.Keyword
			for _, v := range r.Form[f.Name] {
				v = strings.TrimSpace(v)
				if v != "" {
					keywords = append(keywords, bbl.Keyword{Val: v})
				}
			}
			if len(keywords) > 0 {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: keywords})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "extent":
			start := strings.TrimSpace(r.FormValue(f.Name + ".start"))
			end := strings.TrimSpace(r.FormValue(f.Name + ".end"))
			if start != "" || end != "" {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: bbl.Extent{Start: start, End: end}})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "conference":
			name := strings.TrimSpace(r.FormValue(f.Name + ".name"))
			organizer := strings.TrimSpace(r.FormValue(f.Name + ".organizer"))
			location := strings.TrimSpace(r.FormValue(f.Name + ".location"))
			if name != "" || organizer != "" || location != "" {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: bbl.Conference{Name: name, Organizer: organizer, Location: location}})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "note":
			kinds := r.Form[f.Name+".kind"]
			vals := r.Form[f.Name+".val"]
			var notes []bbl.Note
			for i := range min(len(kinds), len(vals)) {
				kind := strings.TrimSpace(kinds[i])
				val := strings.TrimSpace(vals[i])
				if val != "" {
					notes = append(notes, bbl.Note{Kind: kind, Val: val})
				}
			}
			if len(notes) > 0 {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: f.Name, Val: notes})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: f.Name})
			}
		case "workContributor":
			var contributors []bbl.WorkContributor
			for _, g := range formGroups(r.Form, "contributors") {
				name := strings.TrimSpace(g.Get("name"))
				gn := strings.TrimSpace(g.Get("given_name"))
				fn := strings.TrimSpace(g.Get("family_name"))
				if name == "" {
					name = strings.TrimSpace(gn + " " + fn)
				}
				if name == "" {
					continue
				}
				co := bbl.WorkContributor{
					Kind:       g.Get("kind"),
					Name:       name,
					GivenName:  gn,
					FamilyName: fn,
					Roles:      g["roles"],
				}
				if pid := g.Get("person_id"); pid != "" {
					if id, err := bbl.ParseID(pid); err == nil {
						co.PersonID = &id
					}
				}
				contributors = append(contributors, co)
			}
			if len(contributors) > 0 {
				updates = append(updates, &bbl.Set{RecordType: "work", RecordID: work.ID, Field: "contributors", Val: contributors})
			} else {
				updates = append(updates, &bbl.Unset{RecordType: "work", RecordID: work.ID, Field: "contributors"})
			}
		}
	}
	return updates
}
