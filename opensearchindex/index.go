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
}

func New(ctx context.Context, client *opensearchapi.Client) (*Index, error) {
	organizationsIndex, err := newRecIndex(ctx, client, "bbl_organizations", strings.NewReader(organizationSettings), organizationToDoc, generateOrganizationQuery)
	if err != nil {
		return nil, err
	}
	peopleIndex, err := newRecIndex(ctx, client, "bbl_people", strings.NewReader(personSettings), personToDoc, generatePersonQuery)
	if err != nil {
		return nil, err
	}

	return &Index{
		organizationsIndex: organizationsIndex,
		peopleIndex:        peopleIndex,
	}, nil
}

func (idx *Index) Organizations() bbl.RecIndex[*bbl.Organization] {
	return idx.organizationsIndex
}

func (idx *Index) People() bbl.RecIndex[*bbl.Person] {
	return idx.peopleIndex
}

type recIndex[T bbl.Rec] struct {
	client        *opensearchapi.Client
	alias         string
	retention     int
	toDoc         func(T) any
	generateQuery func(string) (string, error)
	bulkIndexer   opensearchutil.BulkIndexer
}

func newRecIndex[T bbl.Rec](ctx context.Context, client *opensearchapi.Client, alias string, settings io.ReadSeeker, toDoc func(T) any, generateQuery func(string) (string, error)) (*recIndex[T], error) {
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

func (idx *recIndex[T]) Search(ctx context.Context, args bbl.SearchArgs) (*bbl.RecHits[T], error) {
	query := `{"match_all": {}}`
	sort := `{"_id": "asc"}`
	searchAfter := ``

	if args.Query != "" {
		q, err := idx.generateQuery(args.Query)
		if err != nil {
			return nil, err
		}
		query = q

		sort = `[{"_score": "desc"}, {"_id": "asc"}]`
	}

	if args.Cursor != "" {
		cursor, err := base64.StdEncoding.DecodeString(args.Cursor)
		if err != nil {
			return nil, err
		}
		searchAfter = `"search_after": ` + string(cursor) + `,`
	}

	body := `{
		"query": ` + query + `,
		"sort": ` + sort + `,
		"size": ` + fmt.Sprint(args.Limit) + `,` +
		searchAfter + `
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

	cursor, err := encodeCursor(res, args)
	if err != nil {
		return nil, err
	}

	hits := &bbl.RecHits[T]{
		Hits:   make([]bbl.RecHit[T], len(res.Hits.Hits)),
		Total:  res.Hits.Total.Value,
		Limit:  args.Limit,
		Query:  args.Query,
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
