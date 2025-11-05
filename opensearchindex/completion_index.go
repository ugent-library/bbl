package opensearchindex

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/opensearchswitcher"
)

//go:embed completion_settings.json
var completionSettings string

type completionDoc struct {
	Completion string `json:"completion"`
}

type completionIndex struct {
	client *opensearchapi.Client
	alias  string
}

func newCompletionIndex(
	ctx context.Context,
	client *opensearchapi.Client,
	alias string,
) (*completionIndex, error) {
	if err := opensearchswitcher.Init(ctx, client, alias, strings.NewReader(completionSettings), 1); err != nil {
		return nil, err
	}

	return &completionIndex{
		client: client,
		alias:  alias,
	}, nil
}

func (idx *completionIndex) NewSwitcher(ctx context.Context) (bbl.IndexSwitcher[string], error) {
	return opensearchswitcher.New(ctx, opensearchswitcher.Config[string]{
		Client:        idx.client,
		Alias:         idx.alias,
		IndexSettings: strings.NewReader(completionSettings),
		Retention:     1,
		ToItem: func(completion string) opensearchswitcher.Item {
			return opensearchswitcher.Item{
				Doc: completionDoc{completion},
			}
		},
	})
}

func (idx *completionIndex) Search(ctx context.Context, q string) (*bbl.CompletionHits, error) {
	jQ, err := jsonString(q)
	if err != nil {
		return nil, err
	}

	body := `{
		"query": {
			"match_phrase_prefix": {
				"completion": "` + jQ + `"
			}
		},
		"size": 10,
		"highlight": {
    		"fields": {
      			"completion": {
					"type": "unified"
				}
    		}
  		}
	}`

	res, err := idx.client.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{idx.alias},
		Body:    strings.NewReader(body),
	})
	if err != nil {
		return nil, err
	}

	hits := &bbl.CompletionHits{
		Hits: make([]bbl.CompletionHit, len(res.Hits.Hits)),
	}
	for i, hit := range res.Hits.Hits {
		var h bbl.CompletionHit
		if err := json.Unmarshal(hit.Source, &h); err != nil {
			return nil, err
		}
		if hl := hit.Highlight["completion"]; len(hl) > 0 {
			h.Highlight.Completion = hl[0]
		}
		hits.Hits[i] = h
	}

	return hits, nil
}
