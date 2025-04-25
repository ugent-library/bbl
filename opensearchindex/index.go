package opensearchindex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/opensearchswitcher"
)

// var versionType = "external"

// assert we implement bbl.Index
var _ bbl.Index = (*Index)(nil)

type Index struct {
	organizationsIndex *recIndex[*bbl.Organization]
	peopleIndex        *recIndex[*bbl.Person]
	projectsIndex      *recIndex[*bbl.Project]
	worksIndex         *recIndex[*bbl.Work]
}

func New(ctx context.Context, client *opensearchapi.Client) (*Index, error) {
	organizationsIndex, err := newRecIndex(ctx, client, "bbl_organizations", strings.NewReader(organizationSettings), organizationToDoc, generateOrganizationQuery, nil)
	if err != nil {
		return nil, err
	}
	peopleIndex, err := newRecIndex(ctx, client, "bbl_people", strings.NewReader(personSettings), personToDoc, generatePersonQuery, nil)
	if err != nil {
		return nil, err
	}
	projectsIndex, err := newRecIndex(ctx, client, "bbl_projects", strings.NewReader(projectSettings), projectToDoc, generateProjectQuery, nil)
	if err != nil {
		return nil, err
	}
	worksIndex, err := newRecIndex(ctx, client, "bbl_works", strings.NewReader(workSettings), workToDoc, generateWorkQuery, generateWorkAggs)
	if err != nil {
		return nil, err
	}

	return &Index{
		organizationsIndex: organizationsIndex,
		peopleIndex:        peopleIndex,
		projectsIndex:      projectsIndex,
		worksIndex:         worksIndex,
	}, nil
}

func (idx *Index) Organizations() bbl.RecIndex[*bbl.Organization] {
	return idx.organizationsIndex
}

func (idx *Index) People() bbl.RecIndex[*bbl.Person] {
	return idx.peopleIndex
}

func (idx *Index) Projects() bbl.RecIndex[*bbl.Project] {
	return idx.projectsIndex
}

func (idx *Index) Works() bbl.RecIndex[*bbl.Work] {
	return idx.worksIndex
}

type recIndex[T bbl.Rec] struct {
	client        *opensearchapi.Client
	alias         string
	retention     int
	toDoc         func(T) any
	generateQuery func(string) (string, error)
	generateAggs  func([]string) (string, error)
	bulkIndexer   opensearchutil.BulkIndexer
}

func newRecIndex[T bbl.Rec](
	ctx context.Context,
	client *opensearchapi.Client,
	alias string,
	settings io.ReadSeeker,
	toDoc func(T) any,
	generateQuery func(string) (string, error),
	generateAggs func([]string) (string, error),
) (*recIndex[T], error) {
	retention := 1

	bulkIndexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client:        client,
		Index:         alias,
		FlushInterval: 1 * time.Second,
		// TODO make configurable
		OnError: func(_ context.Context, err error) {
			log.Printf("error indexing: %s", err)
		},
	})
	if err != nil {
		return nil, err
	}

	if err := opensearchswitcher.Init(ctx, client, alias, settings, retention); err != nil {
		return nil, err
	}

	return &recIndex[T]{
		client:        client,
		alias:         alias,
		retention:     retention,
		bulkIndexer:   bulkIndexer,
		toDoc:         toDoc,
		generateQuery: generateQuery,
		generateAggs:  generateAggs,
	}, nil
}

func (idx *recIndex[T]) NewSwitcher(ctx context.Context) (bbl.RecIndexSwitcher[T], error) {
	return opensearchswitcher.New(ctx, opensearchswitcher.Config[T]{
		Client:        idx.client,
		Alias:         idx.alias,
		IndexSettings: strings.NewReader(personSettings),
		Retention:     idx.retention,
		ToItem: func(rec T) opensearchswitcher.Item {
			// TODO pass version
			return opensearchswitcher.Item{
				Doc: idx.toDoc(rec),
				ID:  rec.RecID(),
			}
		},
	})
}

func (idx *recIndex[T]) Add(ctx context.Context, rec T) error {
	b, err := json.Marshal(idx.toDoc(rec))
	if err != nil {
		return err
	}

	err = idx.bulkIndexer.Add(ctx, opensearchutil.BulkIndexerItem{
		Action:     "index",
		DocumentID: rec.RecID(),
		// TODO
		// Version:     rec.RecVersion(),
		// VersionType: &versionType,
		Body: bytes.NewReader(b),
		// TODO make configurable
		OnFailure: func(_ context.Context, biItem opensearchutil.BulkIndexerItem, _ opensearchapi.BulkRespItem, err error) {
			log.Printf("error indexing %s: %s", biItem.DocumentID, err)
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (idx *recIndex[T]) Search(ctx context.Context, opts bbl.SearchOpts) (*bbl.RecHits[T], error) {
	query := `{"match_all": {}}`
	sort := `{"_id": "asc"}`
	aggs := ``
	paging := ``

	if opts.Query != "" {
		q, err := idx.generateQuery(opts.Query)
		if err != nil {
			return nil, err
		}
		query = q

		sort = `[{"_score": "desc"}, {"_id": "asc"}]`
	}

	// TODO remove nil check
	if idx.generateAggs != nil && len(opts.Facets) > 0 {
		a, err := idx.generateAggs(opts.Facets)
		if err != nil {
			return nil, err
		}
		aggs = a
	}

	if opts.From != 0 {
		paging = `"from": ` + fmt.Sprint(opts.Size) + `,`
	} else if opts.Cursor != "" {
		cursor, err := base64.StdEncoding.DecodeString(opts.Cursor)
		if err != nil {
			return nil, err
		}
		paging = `"search_after": ` + string(cursor) + `,`
	}

	body := `{
		"query": ` + query + `,
		"sort": ` + sort + `,
		"size": ` + fmt.Sprint(opts.Size) + `,` +
		aggs +
		paging + `
		"_source": {
			"includes": ["rec"]
		}
	}`

	res, err := idx.client.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{idx.alias},
		Body:    strings.NewReader(body),
	})
	if err != nil {
		return nil, err
	}

	cursor, err := encodeCursor(res, opts)
	if err != nil {
		return nil, err
	}

	hits := &bbl.RecHits[T]{
		Hits:   make([]bbl.RecHit[T], len(res.Hits.Hits)),
		Total:  res.Hits.Total.Value,
		Query:  opts.Query,
		Size:   opts.Size,
		From:   opts.From,
		Cursor: cursor,
	}
	for i, hit := range res.Hits.Hits {
		var src struct {
			Rec T `json:"rec"`
		}
		if err := json.Unmarshal(hit.Source, &src); err != nil {
			return nil, err
		}
		hits.Hits[i].Rec = src.Rec
	}

	return hits, nil
}
