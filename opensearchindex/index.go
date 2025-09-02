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
	worksIndex, err := newRecIndex(ctx, client, "bbl_works", workSettings, workToDoc, generateWorkQuery, workTermsFilters, generateWorkAggs)
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
	settings      string
	retention     int
	toDoc         func(T) any
	generateQuery func(string) (string, error)
	termsFilters  map[string]string
	generateAggs  func([]string) (map[string]string, error)
	bulkIndexer   opensearchutil.BulkIndexer
}

func newRecIndex[T bbl.Rec](
	ctx context.Context,
	client *opensearchapi.Client,
	alias string,
	settings string,
	toDoc func(T) any,
	generateQuery func(string) (string, error),
	termsFilters map[string]string,
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
		client:        client,
		alias:         alias,
		settings:      settings,
		retention:     retention,
		bulkIndexer:   bulkIndexer,
		toDoc:         toDoc,
		generateQuery: generateQuery,
		termsFilters:  termsFilters,
		generateAggs:  generateAggs,
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

// TODO ErrNotFound
func (idx *recIndex[T]) Get(ctx context.Context, id string) (T, error) {
	var src struct {
		Rec T `json:"rec"`
	}

	res, err := idx.client.Document.Get(ctx, opensearchapi.DocumentGetReq{
		Index:      idx.alias,
		DocumentID: id,
		Params: opensearchapi.DocumentGetParams{
			SourceIncludes: []string{"rec"},
		},
	})

	if err != nil {
		if res.Inspect().Response.StatusCode == 404 {
			return src.Rec, bbl.ErrNotFound
		}
		return src.Rec, err
	}

	err = json.Unmarshal(res.Source, &src)

	return src.Rec, err
}

func (idx *recIndex[T]) Search(ctx context.Context, opts *bbl.SearchOpts) (*bbl.RecHits[T], error) {
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

	if opts.QueryFilter != nil {
		jFilter, err := generateAndFilter(opts.QueryFilter.And, idx.termsFilters)
		if err != nil {
			return nil, err
		}
		query, err = sjson.SetRaw(query, "bool.filter", jFilter)
		if err != nil {
			return nil, err
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
			jFacet := `{"aggs": {"facet": ` + facet + `}}`
			jFacet, err = sjson.SetRaw(jFacet, "filter", query)
			if err != nil {
				return nil, err
			}

			// the facet filter is the query except the terms filter matching the facet
			if opts.QueryFilter != nil {
				for i, f := range opts.QueryFilter.And {
					if f.Terms != nil && f.Terms.Field == key {
						jFacet, err = sjson.Delete(jFacet, "filter.bool.filter."+fmt.Sprint(i))
						if err != nil {
							return nil, err
						}
						break
					}
				}
			}

			facets, err = sjson.SetRaw(facets, key, jFacet)
			if err != nil {
				return nil, err
			}
		}

		aggs = `
		"aggs": {
			"facets": {
				"global": {},
				"aggs": ` + facets + `
			}
		},`
	}

	if opts.Cursor != "" {
		cursor, err := base64.StdEncoding.DecodeString(opts.Cursor)
		if err != nil {
			return nil, err
		}
		paging = `"search_after": ` + string(cursor) + `,`
	} else if opts.From != 0 {
		paging = `"from": ` + fmt.Sprint(opts.From) + `,`
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
		Opts:   opts,
		Hits:   make([]bbl.RecHit[T], len(res.Hits.Hits)),
		Total:  res.Hits.Total.Value,
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

func generateAndFilter(filters []*bbl.AndFilter, termsFilters map[string]string) (string, error) {
	jFilters := `[]`

	for _, filter := range filters {
		var jFilter string
		var err error

		if filter.Or != nil {
			jFilter, err = generateOrFilter(filter.Or, termsFilters)
			if err != nil {
				return jFilters, err
			}
		} else if filter.Terms != nil {
			jFilter, err = generateTermsFilter(filter.Terms, termsFilters)
			if err != nil {
				return jFilters, err
			}
		}

		jFilters, err = sjson.SetRaw(jFilters, "-1", jFilter)
		if err != nil {
			return jFilters, err
		}
	}

	return sjson.SetRaw(``, "bool.must", jFilters)
}

func generateOrFilter(filters []*bbl.OrFilter, termsFilters map[string]string) (string, error) {
	jFilters := `[]`

	for _, filter := range filters {
		var jFilter string
		var err error

		if filter.And != nil {
			jFilter, err = generateAndFilter(filter.And, termsFilters)
			if err != nil {
				return jFilters, err
			}
		} else if filter.Terms != nil {
			jFilter, err = generateTermsFilter(filter.Terms, termsFilters)
			if err != nil {
				return jFilters, err
			}
		}

		jFilters, err = sjson.SetRaw(jFilters, "-1", jFilter)
		if err != nil {
			return jFilters, err
		}
	}

	return sjson.SetRaw(``, "bool.should", jFilters)
}

func generateTermsFilter(filter *bbl.TermsFilter, termsFilters map[string]string) (string, error) {
	docField, ok := termsFilters[filter.Field]
	if !ok {
		return "", fmt.Errorf("unknown terms filter %s", filter.Field)
	}
	return sjson.Set(``, "terms."+docField, filter.Terms)
}
