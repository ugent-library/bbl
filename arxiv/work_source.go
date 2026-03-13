package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/ugent-library/bbl"
)

var reNormalizeID = regexp.MustCompile(`(?i)^arxiv:`)

type feed struct {
	XMLName      xml.Name `xml:"feed"`
	TotalResults int      `xml:"totalResults"`
	Entry        entry    `xml:"entry"`
}

type entry struct {
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Published string `xml:"published"`
	DOI       string `xml:"doi"`
	Comment   string `xml:"comment"`
	Author    []struct {
		Name string `xml:"name"`
	} `xml:"author"`
}

type WorkSource struct {
	url    string
	client *http.Client
}

func NewWorkSource() *WorkSource {
	return &WorkSource{
		url: "https://export.arxiv.org/api/query",
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (ws *WorkSource) Get(ctx context.Context, id string) (*bbl.ImportWorkInput, error) {
	id = reNormalizeID.ReplaceAllString(id, "")

	u, _ := url.Parse(ws.url)
	q := u.Query()
	q.Set("id_list", id)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("arxiv.Get: %w", err)
	}

	res, err := ws.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arxiv.Get: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arxiv.Get: HTTP %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("arxiv.Get: %w", err)
	}

	var f feed
	if err := xml.Unmarshal(body, &f); err != nil {
		return nil, fmt.Errorf("arxiv.Get: %w", err)
	}

	if f.TotalResults != 1 {
		return nil, fmt.Errorf("arxiv.Get: expected 1 result, got %d", f.TotalResults)
	}

	rec := &bbl.ImportWorkInput{
		SourceID:     id,
		Kind:         "journal_article",
		SourceRecord: body,
		Identifiers: []bbl.Identifier{
			{Scheme: "arxiv", Val: id},
		},
		Titles:            []bbl.Title{{Lang: "und", Val: f.Entry.Title}},
		PublicationStatus: "unpublished",
	}

	if f.Entry.DOI != "" {
		rec.Identifiers = append(rec.Identifiers, bbl.Identifier{Scheme: "doi", Val: f.Entry.DOI})
	}

	if f.Entry.Summary != "" {
		rec.Abstracts = append(rec.Abstracts, bbl.Text{Lang: "und", Val: f.Entry.Summary})
	}

	if f.Entry.Comment != "" {
		rec.Notes = append(rec.Notes, bbl.Note{Val: f.Entry.Comment})
	}

	if len(f.Entry.Published) >= 4 {
		rec.PublicationYear = f.Entry.Published[:4]
	}

	for _, a := range f.Entry.Author {
		rec.Contributors = append(rec.Contributors, bbl.ImportWorkContributor{
			Roles: []string{"author"},
			Name:  a.Name,
		})
	}

	return rec, nil
}
