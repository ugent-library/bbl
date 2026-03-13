package views

import (
	"strings"

	"github.com/ugent-library/bbl"
)

// Field getters for template rendering — map field name to Work values.

func workAttrString(work *bbl.Work, name string) string {
	switch name {
	case "article_number":
		return work.ArticleNumber
	case "book_title":
		return work.BookTitle
	case "edition":
		return work.Edition
	case "issue":
		return work.Issue
	case "issue_title":
		return work.IssueTitle
	case "journal_abbreviation":
		return work.JournalAbbreviation
	case "journal_title":
		return work.JournalTitle
	case "place_of_publication":
		return work.PlaceOfPublication
	case "publication_status":
		return work.PublicationStatus
	case "publication_year":
		return work.PublicationYear
	case "publisher":
		return work.Publisher
	case "report_number":
		return work.ReportNumber
	case "series_title":
		return work.SeriesTitle
	case "total_pages":
		return work.TotalPages
	case "volume":
		return work.Volume
	default:
		return ""
	}
}

func workAttrTitleList(work *bbl.Work) []bbl.Title {
	return work.Titles
}

func workAttrTextList(work *bbl.Work, name string) []bbl.Text {
	switch name {
	case "abstracts":
		return work.Abstracts
	case "lay_summaries":
		return work.LaySummaries
	default:
		return nil
	}
}

func workAttrKeywords(work *bbl.Work) []bbl.Keyword {
	return work.Keywords
}

func workAttrExtent(work *bbl.Work, name string) bbl.Extent {
	switch name {
	case "pages":
		return work.Pages
	default:
		return bbl.Extent{}
	}
}

func workAttrConference(work *bbl.Work, name string) bbl.Conference {
	switch name {
	case "conference":
		return work.Conference
	default:
		return bbl.Conference{}
	}
}

func workAttrNoteList(work *bbl.Work, name string) []bbl.Note {
	switch name {
	case "notes":
		return work.Notes
	default:
		return nil
	}
}

func fieldLabel(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "_", " "), ".", " ")
}
