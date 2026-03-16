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

	mutations := buildWorkMutations(r, profile, work)

	if len(mutations) > 0 {
		_, _, err := app.services.Repo.AddRev(r.Context(), bbl.AddRevInput{UserID: userID(c)}, mutations...)
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

// buildWorkMutations builds Set/Delete mutations from the form for all profile fields.
func buildWorkMutations(r *http.Request, profile *bbl.WorkKind, work *bbl.Work) []any {
	var mutations []any
	for _, f := range profile.Fields {
		switch f.Type {
		case "text", "year":
			val := strings.TrimSpace(r.FormValue(f.Name))
			switch f.Name {
			case "article_number":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkArticleNumber{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkArticleNumber{WorkID: work.ID})
				}
			case "book_title":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkBookTitle{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkBookTitle{WorkID: work.ID})
				}
			case "edition":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkEdition{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkEdition{WorkID: work.ID})
				}
			case "issue":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkIssue{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkIssue{WorkID: work.ID})
				}
			case "issue_title":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkIssueTitle{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkIssueTitle{WorkID: work.ID})
				}
			case "journal_abbreviation":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkJournalAbbreviation{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkJournalAbbreviation{WorkID: work.ID})
				}
			case "journal_title":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkJournalTitle{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkJournalTitle{WorkID: work.ID})
				}
			case "place_of_publication":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkPlaceOfPublication{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkPlaceOfPublication{WorkID: work.ID})
				}
			case "publication_status":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkPublicationStatus{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkPublicationStatus{WorkID: work.ID})
				}
			case "publication_year":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkPublicationYear{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkPublicationYear{WorkID: work.ID})
				}
			case "publisher":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkPublisher{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkPublisher{WorkID: work.ID})
				}
			case "report_number":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkReportNumber{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkReportNumber{WorkID: work.ID})
				}
			case "series_title":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkSeriesTitle{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkSeriesTitle{WorkID: work.ID})
				}
			case "total_pages":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkTotalPages{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkTotalPages{WorkID: work.ID})
				}
			case "volume":
				if val != "" {
					mutations = append(mutations, &bbl.SetWorkVolume{WorkID: work.ID, Val: val})
				} else {
					mutations = append(mutations, &bbl.DeleteWorkVolume{WorkID: work.ID})
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
					mutations = append(mutations, &bbl.SetWorkTitles{WorkID: work.ID, Titles: titles})
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
						mutations = append(mutations, &bbl.SetWorkAbstracts{WorkID: work.ID, Abstracts: texts})
					} else {
						mutations = append(mutations, &bbl.DeleteWorkAbstracts{WorkID: work.ID})
					}
				case "lay_summaries":
					if len(texts) > 0 {
						mutations = append(mutations, &bbl.SetWorkLaySummaries{WorkID: work.ID, LaySummaries: texts})
					} else {
						mutations = append(mutations, &bbl.DeleteWorkLaySummaries{WorkID: work.ID})
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
				mutations = append(mutations, &bbl.SetWorkKeywords{WorkID: work.ID, Keywords: keywords})
			} else {
				mutations = append(mutations, &bbl.DeleteWorkKeywords{WorkID: work.ID})
			}
		case "extent":
			start := strings.TrimSpace(r.FormValue(f.Name + ".start"))
			end := strings.TrimSpace(r.FormValue(f.Name + ".end"))
			if start != "" || end != "" {
				mutations = append(mutations, &bbl.SetWorkPages{WorkID: work.ID, Val: bbl.Extent{Start: start, End: end}})
			} else {
				mutations = append(mutations, &bbl.DeleteWorkPages{WorkID: work.ID})
			}
		case "conference":
			name := strings.TrimSpace(r.FormValue(f.Name + ".name"))
			organizer := strings.TrimSpace(r.FormValue(f.Name + ".organizer"))
			location := strings.TrimSpace(r.FormValue(f.Name + ".location"))
			if name != "" || organizer != "" || location != "" {
				mutations = append(mutations, &bbl.SetWorkConference{WorkID: work.ID, Val: bbl.Conference{Name: name, Organizer: organizer, Location: location}})
			} else {
				mutations = append(mutations, &bbl.DeleteWorkConference{WorkID: work.ID})
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
				mutations = append(mutations, &bbl.SetWorkNotes{WorkID: work.ID, Notes: notes})
			} else {
				mutations = append(mutations, &bbl.DeleteWorkNotes{WorkID: work.ID})
			}
		}
	}
	return mutations
}
