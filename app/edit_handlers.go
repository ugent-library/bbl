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
	profile := app.services.Repo.WorkProfiles.Profile(work.Kind)
	if profile == nil {
		return fmt.Errorf("no profile for kind %q", work.Kind)
	}
	return views.BackofficeEditWork(c.ViewCtx, work, profile, nil).Render(r.Context(), w)
}

func (app *App) backofficeUpdateWork(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	work, err := app.getWork(r, "public", "private")
	if err != nil {
		return err
	}
	profile := app.services.Repo.WorkProfiles.Profile(work.Kind)
	if profile == nil {
		return fmt.Errorf("no profile for kind %q", work.Kind)
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	updates := buildWorkUpdates(r, profile, work)

	if len(updates) > 0 {
		_, _, err := app.services.Repo.Update(r.Context(), userID(c), updates...)
		if err != nil {
			return fmt.Errorf("backofficeUpdateWork: %w", err)
		}
	}

	http.Redirect(w, r, fmt.Sprintf("/backoffice/works/%s", work.ID), http.StatusSeeOther)
	return nil
}

func userID(c *Ctx) *bbl.ID {
	if c.User != nil {
		return &c.User.ID
	}
	return nil
}

// buildWorkUpdates builds Set/Delete updates from the form for all profile fields.
func buildWorkUpdates(r *http.Request, profile *bbl.WorkKind, work *bbl.Work) []any {
	var updates []any
	for _, f := range profile.Fields {
		switch f.Type {
		case "text", "year":
			val := strings.TrimSpace(r.FormValue(f.Name))
			switch f.Name {
			case "article_number":
				if val != "" {
					updates = append(updates, &bbl.SetWorkArticleNumber{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkArticleNumber{WorkID: work.ID})
				}
			case "book_title":
				if val != "" {
					updates = append(updates, &bbl.SetWorkBookTitle{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkBookTitle{WorkID: work.ID})
				}
			case "edition":
				if val != "" {
					updates = append(updates, &bbl.SetWorkEdition{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkEdition{WorkID: work.ID})
				}
			case "issue":
				if val != "" {
					updates = append(updates, &bbl.SetWorkIssue{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkIssue{WorkID: work.ID})
				}
			case "issue_title":
				if val != "" {
					updates = append(updates, &bbl.SetWorkIssueTitle{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkIssueTitle{WorkID: work.ID})
				}
			case "journal_abbreviation":
				if val != "" {
					updates = append(updates, &bbl.SetWorkJournalAbbreviation{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkJournalAbbreviation{WorkID: work.ID})
				}
			case "journal_title":
				if val != "" {
					updates = append(updates, &bbl.SetWorkJournalTitle{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkJournalTitle{WorkID: work.ID})
				}
			case "place_of_publication":
				if val != "" {
					updates = append(updates, &bbl.SetWorkPlaceOfPublication{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkPlaceOfPublication{WorkID: work.ID})
				}
			case "publication_status":
				if val != "" {
					updates = append(updates, &bbl.SetWorkPublicationStatus{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkPublicationStatus{WorkID: work.ID})
				}
			case "publication_year":
				if val != "" {
					updates = append(updates, &bbl.SetWorkPublicationYear{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkPublicationYear{WorkID: work.ID})
				}
			case "publisher":
				if val != "" {
					updates = append(updates, &bbl.SetWorkPublisher{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkPublisher{WorkID: work.ID})
				}
			case "report_number":
				if val != "" {
					updates = append(updates, &bbl.SetWorkReportNumber{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkReportNumber{WorkID: work.ID})
				}
			case "series_title":
				if val != "" {
					updates = append(updates, &bbl.SetWorkSeriesTitle{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkSeriesTitle{WorkID: work.ID})
				}
			case "total_pages":
				if val != "" {
					updates = append(updates, &bbl.SetWorkTotalPages{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkTotalPages{WorkID: work.ID})
				}
			case "volume":
				if val != "" {
					updates = append(updates, &bbl.SetWorkVolume{WorkID: work.ID, Val: val})
				} else {
					updates = append(updates, &bbl.UnsetWorkVolume{WorkID: work.ID})
				}
			}
		case "text_list":
			langs := r.Form[f.Name+".lang"]
			vals := r.Form[f.Name+".val"]
			if f.Name == "titles" {
				var titles []bbl.Title
				for i := range min(len(langs), len(vals)) {
					lang := strings.TrimSpace(langs[i])
					val := strings.TrimSpace(vals[i])
					if lang != "" || val != "" {
						titles = append(titles, bbl.Title{Lang: lang, Val: val})
					}
				}
				if len(titles) > 0 {
					updates = append(updates, &bbl.SetWorkTitles{WorkID: work.ID, Titles: titles})
				}
			} else {
				var texts []bbl.Text
				for i := range min(len(langs), len(vals)) {
					lang := strings.TrimSpace(langs[i])
					val := strings.TrimSpace(vals[i])
					if lang != "" || val != "" {
						texts = append(texts, bbl.Text{Lang: lang, Val: val})
					}
				}
				switch f.Name {
				case "abstracts":
					if len(texts) > 0 {
						updates = append(updates, &bbl.SetWorkAbstracts{WorkID: work.ID, Abstracts: texts})
					} else {
						updates = append(updates, &bbl.UnsetWorkAbstracts{WorkID: work.ID})
					}
				case "lay_summaries":
					if len(texts) > 0 {
						updates = append(updates, &bbl.SetWorkLaySummaries{WorkID: work.ID, LaySummaries: texts})
					} else {
						updates = append(updates, &bbl.UnsetWorkLaySummaries{WorkID: work.ID})
					}
				}
			}
		case "string_list":
			var keywords []bbl.Keyword
			for _, v := range r.Form[f.Name] {
				v = strings.TrimSpace(v)
				if v != "" {
					keywords = append(keywords, bbl.Keyword{Val: v})
				}
			}
			if len(keywords) > 0 {
				updates = append(updates, &bbl.SetWorkKeywords{WorkID: work.ID, Keywords: keywords})
			} else {
				updates = append(updates, &bbl.UnsetWorkKeywords{WorkID: work.ID})
			}
		case "extent":
			start := strings.TrimSpace(r.FormValue(f.Name + ".start"))
			end := strings.TrimSpace(r.FormValue(f.Name + ".end"))
			if start != "" || end != "" {
				updates = append(updates, &bbl.SetWorkPages{WorkID: work.ID, Val: bbl.Extent{Start: start, End: end}})
			} else {
				updates = append(updates, &bbl.UnsetWorkPages{WorkID: work.ID})
			}
		case "conference":
			name := strings.TrimSpace(r.FormValue(f.Name + ".name"))
			organizer := strings.TrimSpace(r.FormValue(f.Name + ".organizer"))
			location := strings.TrimSpace(r.FormValue(f.Name + ".location"))
			if name != "" || organizer != "" || location != "" {
				updates = append(updates, &bbl.SetWorkConference{WorkID: work.ID, Val: bbl.Conference{Name: name, Organizer: organizer, Location: location}})
			} else {
				updates = append(updates, &bbl.UnsetWorkConference{WorkID: work.ID})
			}
		case "note_list":
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
				updates = append(updates, &bbl.SetWorkNotes{WorkID: work.ID, Notes: notes})
			} else {
				updates = append(updates, &bbl.UnsetWorkNotes{WorkID: work.ID})
			}
		case "contributor_list":
			var contributors []bbl.WorkContributor
			for i, g := range formGroups(r.Form, "contributors") {
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
					Position:   i,
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
				updates = append(updates, &bbl.SetWorkContributors{WorkID: work.ID, Contributors: contributors})
			} else {
				updates = append(updates, &bbl.UnsetWorkContributors{WorkID: work.ID})
			}
		}
	}
	return updates
}

