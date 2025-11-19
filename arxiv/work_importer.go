package arxiv

import (
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
	Title      string `xml:"title"`
	Summary    string `xml:"summary"`
	Published  string `xml:"published"`
	DOI        string `xml:"doi"`
	JournalRef string `xml:"journal_ref"`
	Author     []struct {
		Name string `xml:"name"`
	} `xml:"author"`
	Comment string `xml:"comment"`
}

type WorkImporter struct {
	url    string
	client *http.Client
}

func NewWorkImporter() *WorkImporter {
	return &WorkImporter{
		url: "https://export.arxiv.org/api/query",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (wi *WorkImporter) Get(id string) (*bbl.Work, error) {
	id = reNormalizeID.ReplaceAllString(id, "")

	u, _ := url.Parse(wi.url)
	q := u.Query()
	q.Set("id_list", id)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("arxiv: request failed: %w", err)
	}
	res, err := wi.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("arxiv: request failed: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arxiv: request failed with status %d", res.StatusCode)
	}

	defer res.Body.Close()
	src, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("arxiv: reading response failed: %w", err)
	}

	f := feed{}

	if err := xml.Unmarshal(src, &f); err != nil {
		return nil, fmt.Errorf("arxiv: unmarshalling response failed: %w", err)
	}

	if f.TotalResults != 1 {
		return nil, fmt.Errorf("arxiv: expected 1 entry, but found %d", f.TotalResults)
	}

	rec := &bbl.Work{
		Header: bbl.Header{
			Identifiers: []bbl.Code{
				{Scheme: "arxiv", Val: id},
			},
		},
		Kind:    "journal_article",
		Subkind: "original",
		WorkAttrs: bbl.WorkAttrs{
			Titles:            []bbl.Text{{Lang: "und", Val: f.Entry.Title}},
			JournalTitle:      f.Entry.JournalRef,
			PublicationStatus: "unpublished",
		},
	}

	if err := bbl.LoadWorkProfile(rec); err != nil {
		return nil, err
	}

	if f.Entry.DOI != "" {
		rec.Identifiers = append(rec.Identifiers, bbl.Code{Scheme: "doi", Val: f.Entry.DOI})
	}

	if f.Entry.Summary != "" {
		rec.Abstracts = append(rec.Abstracts, bbl.Text{Lang: "und", Val: f.Entry.Summary})
	}

	if f.Entry.Comment != "" {
		rec.Notes = append(rec.Notes, bbl.Note{Val: f.Entry.Comment})
	}

	if len(f.Entry.Published) > 4 {
		rec.PublicationYear = f.Entry.Published[0:4]
	} else if len(f.Entry.Published) == 4 {
		rec.PublicationYear = f.Entry.Published
	}

	for _, a := range f.Entry.Author {
		// nameParts := strings.Split(a.Name, " ")
		// firstName := nameParts[0]
		// lastName := nameParts[0]
		// if len(nameParts) > 1 {
		// 	lastName = strings.Join(nameParts[1:], " ")
		// }
		rec.Contributors = append(rec.Contributors, bbl.WorkContributor{
			WorkContributorAttrs: bbl.WorkContributorAttrs{
				CreditRoles: []string{bbl.AuthorCreditRole},
				Name:        a.Name,
				// GivenName:   firstName,
				// FamilyName:  lastName,
			},
		})
	}

	return rec, nil
}
