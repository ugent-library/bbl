package csl

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl"
)

func NewWorkEncoder(citeprocURL, style string) bbl.WorkEncoder {
	return func(rec *bbl.Work) ([]byte, error) {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		b := &bytes.Buffer{}
		err := json.NewEncoder(b).Encode(&requestBody{Items: []item{workToItem(rec)}})
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequest(http.MethodPost, citeprocURL+"?style="+style, b)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, err
		}

		cite := gjson.GetBytes(resBody, "bibliography.1.0").String()

		return []byte(strings.TrimSpace(cite)), nil
	}
}

type requestBody struct {
	Items []item `json:"items"`
}

type item struct {
	ID             string   `json:"id"`
	Type           string   `json:"type,omitempty"`
	Title          string   `json:"title,omitempty"`
	Author         []person `json:"author,omitempty"`
	Edition        string   `json:"edition,omitempty"`
	Issued         issued   `json:"issued,omitempty"`
	Publisher      string   `json:"publisher,omitempty"`
	PublisherPlace string   `json:"publisher-place,omitempty"`
	DOI            string   `json:"DOI,omitempty"`
	ISBN           string   `json:"ISBN,omitempty"`
}

type issued struct {
	Raw string `json:"raw,omitempty"`
}

type person struct {
	Family string `json:"family,omitempty"`
	Given  string `json:"given,omitempty"`
}

func workToItem(rec *bbl.Work) item {
	item := item{
		ID:             rec.ID,
		Title:          rec.GetTitle(),
		Edition:        rec.Edition,
		Publisher:      rec.Publisher,
		PublisherPlace: rec.PlaceOfPublication,
		DOI:            rec.GetIdentifierWithScheme("doi"),
	}

	switch rec.Kind {
	case "book":
		item.Type = "book"
	case "journal_article":
		item.Type = "article-journal"
	case "book_chapter":
		item.Type = "chapter"
	case "dissertation":
		item.Type = "thesis"
	}
	item.Issued.Raw = rec.PublicationYear
	for _, c := range rec.GetContributorsWithCreditRole("author") {
		item.Author = append(item.Author, person{Family: c.FamilyName, Given: c.GivenName})
	}
	if isbn := rec.GetIdentifierWithScheme("isbn"); isbn != "" {
		item.ISBN = isbn
	}

	return item
}
