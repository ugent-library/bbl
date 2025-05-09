package opensearchindex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/opensearchswitcher"
)

var versionType = "external"

// assert we implement bbl.Index
var _ bbl.Index = (*Index)(nil)

type Index struct {
	organizationsIndex *recIndex[*bbl.Organization]
	peopleIndex        *recIndex[*bbl.Person]
	projectsIndex      *recIndex[*bbl.Project]
	worksIndex         *recIndex[*bbl.Work]
}

func New(ctx context.Context, client *opensearchapi.Client) (*Index, error) {
	organizationsIndex, err := newRecIndex(ctx, client, "bbl_organizations", organizationSettings, organizationToDoc, generateOrganizationQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	peopleIndex, err := newRecIndex(ctx, client, "bbl_people", personSettings, personToDoc, generatePersonQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	projectsIndex, err := newRecIndex(ctx, client, "bbl_projects", projectSettings, projectToDoc, generateProjectQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	worksIndex, err := newRecIndex(ctx, client, "bbl_works", workSettings, workToDoc, generateWorkQuery, generateWorkFilters, generateWorkAggs)
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
	client          *opensearchapi.Client
	alias           string
	settings        string
	retention       int
	toDoc           func(T) any
	generateQuery   func(string) (string, error)
	generateFilters func(map[string][]string) (map[string]string, error)
	generateAggs    func([]string) (map[string]string, error)
	bulkIndexer     opensearchutil.BulkIndexer
}

func newRecIndex[T bbl.Rec](
	ctx context.Context,
	client *opensearchapi.Client,
	alias string,
	settings string,
	toDoc func(T) any,
	generateQuery func(string) (string, error),
	generateFilters func(map[string][]string) (map[string]string, error),
	generateAggs func([]string) (map[string]string, error),
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

	if err := opensearchswitcher.Init(ctx, client, alias, strings.NewReader(settings), retention); err != nil {
		return nil, err
	}

	return &recIndex[T]{
		client:          client,
		alias:           alias,
		settings:        settings,
		retention:       retention,
		bulkIndexer:     bulkIndexer,
		toDoc:           toDoc,
		generateQuery:   generateQuery,
		generateFilters: generateFilters,
		generateAggs:    generateAggs,
	}, nil
}

func (idx *recIndex[T]) NewSwitcher(ctx context.Context) (bbl.RecIndexSwitcher[T], error) {
	return opensearchswitcher.New(ctx, opensearchswitcher.Config[T]{
		Client:        idx.client,
		Alias:         idx.alias,
		IndexSettings: strings.NewReader(idx.settings),
		Retention:     idx.retention,
		ToItem: func(rec T) opensearchswitcher.Item {
			hdr := rec.Header()
			return opensearchswitcher.Item{
				Doc:     idx.toDoc(rec),
				ID:      hdr.ID,
				Version: int64(hdr.Version),
			}
		},
	})
}

func (idx *recIndex[T]) Add(ctx context.Context, rec T) error {
	b, err := json.Marshal(idx.toDoc(rec))
	if err != nil {
		return err
	}

	hdr := rec.Header()
	version := int64(hdr.Version)

	err = idx.bulkIndexer.Add(ctx, opensearchutil.BulkIndexerItem{
		Action:      "index",
		DocumentID:  hdr.ID,
		Version:     &version,
		VersionType: &versionType,
		Body:        bytes.NewReader(b),
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
	query := `{
		"bool": {
			"must": [{"match_all": {}}]
		}
	}`
	sort := `{"_id": "asc"}`
	aggs := ``
	paging := ``

	// TODO we assume generateQuery builds a bool query
	if opts.Query != "" {
		q, err := idx.generateQuery(opts.Query)
		if err != nil {
			return nil, err
		}
		query = q

		sort = `[{"_score": "desc"}, {"_id": "asc"}]`
	}

	var filtersMap map[string]string

	// TODO remove nil check
	if idx.generateFilters != nil && len(opts.Filters) > 0 {
		m, err := idx.generateFilters(opts.Filters)
		if err != nil {
			return nil, err
		}
		filtersMap = m

		for _, filter := range filtersMap {
			var err error
			query, err = sjson.SetRaw(query, "bool.filter.-1", filter)
			if err != nil {
				return nil, err
			}
		}
	}

	// TODO remove nil check
	if idx.generateAggs != nil && len(opts.Facets) > 0 {
		m, err := idx.generateAggs(opts.Facets)
		if err != nil {
			return nil, err
		}

		facets := `{}`

		for key, facet := range m {
			f := `{"aggs": {"facet": ` + facet + `}}`

			facetFilter, err := sjson.Delete(query, "bool.filter")
			if err != nil {
				return nil, err
			}
			for filterKey, filter := range filtersMap {
				if filterKey == key {
					continue
				}
				facetFilter, err = sjson.SetRaw(facetFilter, "bool.filter.-1", filter)
				if err != nil {
					return nil, err
				}
			}

			f, err = sjson.SetRaw(f, "filter", facetFilter)
			if err != nil {
				return nil, err
			}

			facets, err = sjson.SetRaw(facets, key, f)
		}

		aggs = `
		"aggs": {
			"facets": {
				"global": {},
				"aggs": ` + facets + `
			}
		},`
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
		Hits:    make([]bbl.RecHit[T], len(res.Hits.Hits)),
		Total:   res.Hits.Total.Value,
		Query:   opts.Query,
		Filters: opts.Filters,
		Size:    opts.Size,
		From:    opts.From,
		Cursor:  cursor,
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

	// TODO remove nil check
	if len(opts.Facets) > 0 && res.Aggregations != nil {
		for _, name := range opts.Facets {
			facet := bbl.Facet{Name: name}
			gjson.GetBytes(res.Aggregations, "facets."+name+".facet.buckets").ForEach(func(k, v gjson.Result) bool {
				facet.Vals = append(facet.Vals, bbl.FacetValue{
					Val:   v.Get("key").String(),
					Count: int(v.Get("doc_count").Int()),
				})
				return true
			})
			if len(facet.Vals) > 0 {
				hits.Facets = append(hits.Facets, facet)
			}
		}
	}

	return hits, nil
}
