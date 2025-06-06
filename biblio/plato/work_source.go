package plato

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl"
)

const count = 100

type WorkSource struct {
	url      *url.URL
	username string
	password string
	client   *http.Client
}

func (ws *WorkSource) Init() error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.BindEnv("plato.url")
	v.BindEnv("plato.username")
	v.BindEnv("plato.password")

	ws.username = v.GetString("plato.username")
	ws.password = v.GetString("plato.password")

	u, err := url.ParseRequestURI(v.GetString("plato.url"))
	if err != nil {
		return err
	}
	ws.url = u

	ws.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	return nil
}

func (ws *WorkSource) Interval() time.Duration {
	return 24 * time.Hour
}

func (ws *WorkSource) MatchIdentifierScheme() string {
	return "plato"
}

func (ws *WorkSource) Iter(ctx context.Context) iter.Seq2[*bbl.Work, error] {
	return func(yield func(*bbl.Work, error) bool) {
		for from := 1; ; from += count {
			u := *ws.url
			q := u.Query()
			q.Set("from", fmt.Sprint(from))
			q.Set("count", fmt.Sprint(count))
			u.RawQuery = q.Encode()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				yield(nil, err)
				return
			}
			req.SetBasicAuth(ws.username, ws.password)
			res, err := ws.client.Do(req)
			if err != nil {
				yield(nil, err)
				return
			}
			if res.StatusCode < 200 || res.StatusCode >= 400 {
				yield(nil, fmt.Errorf("GET %q: %s", u.String(), res.Status))
				return
			}

			body, err := io.ReadAll(res.Body)
			if err != nil {
				yield(nil, err)
				return
			}

			list := gjson.GetBytes(body, "list").Array()

			for _, data := range list {
				if !yield(mapWork(data)) {
					return
				}
			}

			if len(list) < count {
				return
			}
		}
	}
}

func mapWork(res gjson.Result) (*bbl.Work, error) {
	platoID := res.Get("plato_id").String()

	rec := &bbl.Work{
		Kind:   "dissertation",
		Status: bbl.SuggestionStatus,
		Identifiers: []bbl.Code{
			{Scheme: "plato", Val: platoID},
		},
		Attrs: bbl.WorkAttrs{
			Classifications: []bbl.Code{
				{Scheme: "ugent_classification", Val: "U"},
			},
			PlaceOfPublication: "Ghent, Belgium",
		},
	}

	if v := res.Get("titel.eng").String(); v != "" {
		rec.Attrs.Titles = append(rec.Attrs.Titles, bbl.Text{Lang: "eng", Val: v})
	}
	if v := res.Get("titel.ned").String(); v != "" {
		rec.Attrs.Titles = append(rec.Attrs.Titles, bbl.Text{Lang: "dut", Val: v})
	}

	// TODO
	// p.PublicationStatus = "published"

	// TODO
	// if v := md.Get("defence.date").String(); v != "" {
	// 	p.DefenseDate = v
	// }
	// p.DefensePlace = "Ghent, Belgium"

	// TODO
	// ugentID := md.Get("student.ugentid").String()
	// if ugentID == "" && md.Get("student.studid").String() != "" {
	// 	ugentID = "0000" + md.Get("student.studid").String()
	// }
	// if ugentID != "" {
	// 	hits, err := services.PersonSearchService.SuggestPeople(ugentID)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if len(hits) == 0 {
	// 		return nil, errors.New("no matches for ugent id " + ugentID)
	// 	}
	// 	c := models.ContributorFromPerson(hits[0])
	// 	p.Author = append(p.Author, c)
	// } else {
	// 	c := models.ContributorFromFirstLastName(md.Get("student.first").String(), md.Get("student.last").String())
	// 	c.ExternalPerson.Affiliation = md.Get("student.affil").String()
	// 	c.ExternalPerson.HonorificPrefix = md.Get("student.title").String()
	// 	p.Author = append(p.Author, c)
	// }

	// TODO
	// var cbErr error
	// md.Get("supervisors").ForEach(func(key, val gjson.Result) bool {
	// 	if v := val.Get("ugentid").String(); v != "" {
	// 		hits, err := services.PersonSearchService.SuggestPeople(v)
	// 		if err != nil {
	// 			cbErr = err
	// 			return false
	// 		}
	// 		if len(hits) == 0 {
	// 			cbErr = errors.New("no matches for ugent id " + v)
	// 			return false
	// 		}
	// 		p.Supervisor = append(p.Supervisor, models.ContributorFromPerson(hits[0]))

	// 		for _, aff := range hits[0].Affiliations {
	// 			p.RemoveOrganization(aff.OrganizationID)
	// 			p.RelatedOrganizations = append(p.RelatedOrganizations, &models.RelatedOrganization{
	// 				OrganizationID: aff.OrganizationID,
	// 			})
	// 		}
	// 	} else {
	// 		c := models.ContributorFromFirstLastName(val.Get("first").String(), val.Get("last").String())
	// 		c.ExternalPerson.Affiliation = val.Get("affil").String()
	// 		c.ExternalPerson.HonorificPrefix = val.Get("title").String()
	// 		p.Supervisor = append(p.Supervisor, c)
	// 	}
	// 	return true
	// })
	// if cbErr != nil {
	// 	return nil, cbErr
	// }

	if v := res.Get("pdf.ISBN").String(); v != "" {
		rec.Identifiers = append(rec.Identifiers, bbl.Code{Scheme: "isbn", Val: v})
	}

	if v := res.Get("pdf.abstract").String(); v != "" {
		rec.Attrs.Abstracts = append(rec.Attrs.Abstracts, bbl.Text{Lang: "dut", Val: v})
	}

	// TODO
	// if v := res.Get("pdf.confidential_reason").String(); v != "" {
	// 	p.ReviewerNote = fmt.Sprintf("plato confidential reason: %s", v)
	// }

	// TODO
	// if v := md.Get("pdf.url").String(); v != "" {
	// 	sha256, size, err := recordsources.StoreURL(context.TODO(), v, services.FileStore)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	f := &models.PublicationFile{
	// 		Relation:           "main_file",
	// 		Name:               r.id + ".pdf",
	// 		ContentType:        "application/pdf",
	// 		Size:               size,
	// 		SHA256:             sha256,
	// 		PublicationVersion: "publishedVersion",
	// 	}
	// 	embargo := md.Get("pdf.embargo").String()
	// 	access := md.Get("pdf.accesstype").String()
	// 	if strings.HasPrefix(embargo, "9999") {
	// 		f.AccessLevel = "info:eu-repo/semantics/closedAccess"
	// 	} else if embargo != "" {
	// 		f.AccessLevel = "info:eu-repo/semantics/embargoedAccess"
	// 		f.AccessLevelDuringEmbargo = "info:eu-repo/semantics/closedAccess"
	// 		f.EmbargoDate = embargo[:10]
	// 		if access == "U" {
	// 			f.AccessLevelAfterEmbargo = "info:eu-repo/semantics/restrictedAccess"
	// 		} else if access == "W" {
	// 			f.AccessLevelAfterEmbargo = "info:eu-repo/semantics/openAccess"
	// 		}
	// 	} else if access == "U" {
	// 		f.AccessLevel = "info:eu-repo/semantics/restrictedAccess"
	// 	} else if access == "W" {
	// 		f.AccessLevel = "info:eu-repo/semantics/openAccess"
	// 	}
	// 	p.AddFile(f)
	// }

	return rec, nil
}
