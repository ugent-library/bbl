// Package citeformat formats works as citations using a citeproc-js-server.
package citeformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ugent-library/bbl"
)

// Client calls a citeproc-js-server to format citations.
type Client struct {
	URL   string // base URL of the citeproc-js-server (e.g. "http://localhost:8085")
	Style string // CSL style name (e.g. "apa", "chicago-author-date")
}

// Format formats a single work as a citation string.
func (c *Client) Format(work *bbl.Work) (string, error) {
	item := workToCSLItem(work)
	body := request{
		Items:   map[string]cslItem{item.ID: item},
		ItemIDs: []string{item.ID},
	}

	b, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("citeproc: marshal: %w", err)
	}

	url := strings.TrimRight(c.URL, "/") + "?responseformat=html&style=" + c.Style
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("citeproc: request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("citeproc: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("citeproc: HTTP %d: %s", resp.StatusCode, respBody)
	}

	return string(respBody), nil
}

type request struct {
	Items   map[string]cslItem `json:"items"`
	ItemIDs []string           `json:"itemIDs"`
}

type cslItem struct {
	ID             string      `json:"id"`
	Type           string      `json:"type"`
	Title          string      `json:"title,omitempty"`
	Author         []cslName   `json:"author,omitempty"`
	Editor         []cslName   `json:"editor,omitempty"`
	Issued         *cslDate    `json:"issued,omitempty"`
	ContainerTitle string      `json:"container-title,omitempty"`
	Volume         string      `json:"volume,omitempty"`
	Issue          string      `json:"issue,omitempty"`
	Page           string      `json:"page,omitempty"`
	Publisher      string      `json:"publisher,omitempty"`
	PublisherPlace string      `json:"publisher-place,omitempty"`
	Edition        string      `json:"edition,omitempty"`
	DOI            string      `json:"DOI,omitempty"`
	ISBN           string      `json:"ISBN,omitempty"`
	ISSN           string      `json:"ISSN,omitempty"`
	URL            string      `json:"URL,omitempty"`
	CollectionTitle string     `json:"collection-title,omitempty"`
	EventTitle     string      `json:"event-title,omitempty"`
	EventPlace     string      `json:"event-place,omitempty"`
	Number         string      `json:"number,omitempty"`
	NumberOfPages  string      `json:"number-of-pages,omitempty"`
	ArticleNumber  string      `json:"article-number,omitempty"`  // custom, some styles support this
	Abstract       string      `json:"abstract,omitempty"`
}

type cslName struct {
	Family  string `json:"family,omitempty"`
	Given   string `json:"given,omitempty"`
	Literal string `json:"literal,omitempty"`
}

type cslDate struct {
	DateParts [][]string `json:"date-parts"`
}

// workKindToCSLType maps bbl work kinds to CSL types.
// Unmapped kinds fall back to "document".
var workKindToCSLType = map[string]string{
	"journal_article":  "article-journal",
	"book":             "book",
	"book_chapter":     "chapter",
	"conference_paper": "paper-conference",
	"dissertation":     "thesis",
	"report":           "report",
	"preprint":         "article",
	"book_editor":      "book",
	"issue_editor":     "book",
}

func workToCSLItem(w *bbl.Work) cslItem {
	item := cslItem{
		ID:             w.ID.String(),
		Type:           workKindToCSLType[w.Kind],
		Volume:         w.Volume,
		Issue:          w.Issue,
		Publisher:      w.Publisher,
		PublisherPlace: w.PlaceOfPublication,
		Edition:        w.Edition,
		CollectionTitle: w.SeriesTitle,
		NumberOfPages:  w.TotalPages,
		ArticleNumber:  w.ArticleNumber,
		Number:         w.ReportNumber,
	}

	if item.Type == "" {
		item.Type = "document"
	}

	// Title: use the first title.
	if len(w.Titles) > 0 {
		item.Title = w.Titles[0].Val
	}

	// Container title: journal or book title depending on kind.
	switch w.Kind {
	case "journal_article":
		item.ContainerTitle = w.JournalTitle
	case "book_chapter":
		item.ContainerTitle = w.BookTitle
	}

	// Pages.
	if w.Pages.Start != "" {
		if w.Pages.End != "" {
			item.Page = w.Pages.Start + "-" + w.Pages.End
		} else {
			item.Page = w.Pages.Start
		}
	}

	// Publication year.
	if w.PublicationYear != "" {
		item.Issued = &cslDate{DateParts: [][]string{{w.PublicationYear}}}
	}

	// Conference.
	if w.Conference.Name != "" {
		item.EventTitle = w.Conference.Name
	}
	if w.Conference.Location != "" {
		item.EventPlace = w.Conference.Location
	}

	// Contributors: split by role.
	for _, c := range w.Contributors {
		name := contributorToCSLName(c)
		isEditor := false
		for _, r := range c.Roles {
			if r == "editor" {
				isEditor = true
				break
			}
		}
		if isEditor {
			item.Editor = append(item.Editor, name)
		} else {
			item.Author = append(item.Author, name)
		}
	}

	// Identifiers.
	for _, id := range w.Identifiers {
		switch id.Scheme {
		case "doi":
			if item.DOI == "" {
				item.DOI = id.Val
			}
		case "isbn":
			if item.ISBN == "" {
				item.ISBN = id.Val
			}
		case "issn":
			if item.ISSN == "" {
				item.ISSN = id.Val
			}
		case "url":
			if item.URL == "" {
				item.URL = id.Val
			}
		}
	}

	// Abstract: use the first abstract.
	if len(w.Abstracts) > 0 {
		item.Abstract = w.Abstracts[0].Val
	}

	return item
}

func contributorToCSLName(c bbl.WorkContributor) cslName {
	if c.Name != "" {
		return cslName{Literal: c.Name}
	}
	return cslName{
		Family: c.FamilyName,
		Given:  c.GivenName,
	}
}
