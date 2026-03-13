package plato

import (
	"context"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"time"

	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl"
)

const pageSize = 100

type Config struct {
	URL      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type WorkSource struct {
	url      *url.URL
	username string
	password string
	client   *http.Client
}

func New(c Config) (bbl.WorkSourceIter, error) {
	u, err := url.ParseRequestURI(c.URL)
	if err != nil {
		return nil, err
	}
	return &WorkSource{
		url:      u,
		username: c.Username,
		password: c.Password,
		client:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (ws *WorkSource) Iter(ctx context.Context) (iter.Seq2[*bbl.ImportWorkInput, error], error) {
	seq := func(yield func(*bbl.ImportWorkInput, error) bool) {
		for from := 1; ; from += pageSize {
			u := *ws.url
			q := u.Query()
			q.Set("from", fmt.Sprint(from))
			q.Set("count", fmt.Sprint(pageSize))
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

			if res.StatusCode < 200 || res.StatusCode >= 300 {
				res.Body.Close()
				yield(nil, fmt.Errorf("GET %q: %s", u.String(), res.Status))
				return
			}

			body, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				yield(nil, err)
				return
			}

			list := gjson.GetBytes(body, "list").Array()

			for _, data := range list {
				rec := mapWork(data)
				if !yield(rec, nil) {
					return
				}
			}

			if len(list) < pageSize {
				return
			}
		}
	}
	return seq, nil
}

func mapWork(res gjson.Result) *bbl.ImportWorkInput {
	platoID := res.Get("plato_id").String()

	rec := &bbl.ImportWorkInput{
		SourceID: platoID,
		Kind:     "dissertation",
		Status:   "private",
		Identifiers: []bbl.Identifier{
			{Scheme: "plato_id", Val: platoID},
		},
		Classifications: []bbl.Identifier{
			{Scheme: "ugent", Val: "U"},
		},
	}

	if v := res.Get("titel.eng").String(); v != "" {
		rec.Titles = append(rec.Titles, bbl.Title{Lang: "eng", Val: v})
	}
	if v := res.Get("titel.ned").String(); v != "" {
		rec.Titles = append(rec.Titles, bbl.Title{Lang: "dut", Val: v})
	}

	rec.PlaceOfPublication = "Ghent, Belgium"

	if v := res.Get("pdf.ISBN").String(); v != "" {
		rec.Identifiers = append(rec.Identifiers, bbl.Identifier{Scheme: "isbn", Val: v})
	}

	if v := res.Get("pdf.abstract").String(); v != "" {
		rec.Abstracts = append(rec.Abstracts, bbl.Text{Lang: "dut", Val: v})
	}

	return rec
}
