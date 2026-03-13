package opensearchindex

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/opensearch-project/opensearch-go/v4/opensearchutil"
	"github.com/ugent-library/bbl"
)

const maxOffset = 10000

var (
	versionType = "external"
	refreshTrue = true
)

// facetDef describes a facet for an entity type.
type facetDef struct {
	Field string
	Size  int
}

// searchIndex is the generic OpenSearch implementation for a single entity type.
// T is the domain entity type (e.g. *bbl.Work), H is the hit type (e.g. bbl.WorkHit).
type searchIndex[T any, H any] struct {
	client     *opensearchapi.Client
	alias      string
	settings   string
	toDoc      func(T) (id string, version int, doc map[string]any)
	toHit      func(id string, doc map[string]any) H
	buildQuery func(string) map[string]any
	facetDefs  map[string]facetDef
	filterDefs map[string]string // logical name -> doc field path
	onFail     func(ctx context.Context, id string, err error)
}

type searchHits[H any] struct {
	Hits   []H
	Total  int
	Cursor string
	Facets []bbl.Facet
}

func (idx *searchIndex[T, H]) add(ctx context.Context, entity T) error {
	id, version, doc := idx.toDoc(entity)

	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("opensearchindex: marshal doc %s: %w", id, err)
	}

	ver := version
	_, err = idx.client.Index(ctx, opensearchapi.IndexReq{
		Index:      idx.alias,
		DocumentID: id,
		Body:       bytes.NewReader(b),
		Params: opensearchapi.IndexParams{
			Version:     &ver,
			VersionType: versionType,
		},
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: index %s: %w", id, err)
	}

	return nil
}

func (idx *searchIndex[T, H]) bulkAdd(ctx context.Context, bulkIndexer opensearchutil.BulkIndexer, entity T) error {
	id, version, doc := idx.toDoc(entity)

	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("opensearchindex: marshal doc %s: %w", id, err)
	}

	ver := int64(version)
	return bulkIndexer.Add(ctx, opensearchutil.BulkIndexerItem{
		Action:      "index",
		DocumentID:  id,
		Version:     &ver,
		VersionType: &versionType,
		Body:        bytes.NewReader(b),
		OnFailure: func(_ context.Context, item opensearchutil.BulkIndexerItem, _ opensearchapi.BulkRespItem, err error) {
			if idx.onFail != nil {
				idx.onFail(ctx, item.DocumentID, err)
			}
		},
	})
}

func (idx *searchIndex[T, H]) delete(ctx context.Context, id bbl.ID) error {
	_, err := idx.client.Document.Delete(ctx, opensearchapi.DocumentDeleteReq{
		Index:      idx.alias,
		DocumentID: id.String(),
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: delete %s: %w", id, err)
	}
	return nil
}

func (idx *searchIndex[T, H]) deleteAll(ctx context.Context) error {
	body, _ := json.Marshal(map[string]any{
		"query": map[string]any{"match_all": map[string]any{}},
	})
	_, err := idx.client.Document.DeleteByQuery(ctx, opensearchapi.DocumentDeleteByQueryReq{
		Indices: []string{idx.alias},
		Body:    bytes.NewReader(body),
		Params: opensearchapi.DocumentDeleteByQueryParams{
			Refresh: &refreshTrue,
		},
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: delete all %s: %w", idx.alias, err)
	}
	return nil
}

func (idx *searchIndex[T, H]) search(ctx context.Context, opts *bbl.SearchOpts) (*searchHits[H], error) {
	// Build query.
	var query map[string]any
	if opts.Query != "" {
		query = idx.buildQuery(opts.Query)
	} else {
		query = boolQuery(must(matchAll()))
	}

	// Apply filters.
	if opts.Filter != nil {
		filterClauses, err := idx.buildFilterClauses(opts.Filter.And)
		if err != nil {
			return nil, err
		}
		// Add filter to bool query.
		if b, ok := query["bool"].(map[string]any); ok {
			b["filter"] = filterClauses
		}
	}

	// Sort: by score+id when searching, by id only when browsing.
	var sort any
	if opts.Query != "" {
		sort = []any{
			map[string]any{"_score": "desc"},
			map[string]any{"id": "asc"},
		}
	} else {
		sort = []any{
			map[string]any{"id": "asc"},
		}
	}

	// Build request body.
	body := map[string]any{
		"query": query,
		"sort":  sort,
		"size":  opts.Size,
	}

	// Pagination.
	if opts.Cursor != "" {
		cursor, err := base64.StdEncoding.DecodeString(opts.Cursor)
		if err != nil {
			return nil, fmt.Errorf("opensearchindex: invalid cursor: %w", err)
		}
		var searchAfter []any
		if err := json.Unmarshal(cursor, &searchAfter); err != nil {
			return nil, fmt.Errorf("opensearchindex: invalid cursor: %w", err)
		}
		body["search_after"] = searchAfter
	} else if opts.Offset > 0 {
		offset := opts.Offset
		if offset > maxOffset {
			offset = maxOffset
		}
		body["from"] = offset
	}

	// Facets (global aggregations with per-facet filter exclusion).
	if len(opts.Facets) > 0 {
		aggs, err := idx.buildFacetAggs(query, opts)
		if err != nil {
			return nil, err
		}
		if len(aggs) > 0 {
			body["aggs"] = map[string]any{
				"facets": map[string]any{
					"global": map[string]any{},
					"aggs":   aggs,
				},
			}
		}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("opensearchindex: marshal search body: %w", err)
	}

	res, err := idx.client.Search(ctx, &opensearchapi.SearchReq{
		Indices: []string{idx.alias},
		Body:    bytes.NewReader(bodyBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("opensearchindex: search: %w", err)
	}

	// Build cursor from last hit's sort values.
	cursor := ""
	if n := len(res.Hits.Hits); n > 0 && n >= opts.Size {
		sortBytes, err := json.Marshal(res.Hits.Hits[n-1].Sort)
		if err != nil {
			return nil, fmt.Errorf("opensearchindex: encode cursor: %w", err)
		}
		cursor = base64.StdEncoding.EncodeToString(sortBytes)
	}

	// Parse hits.
	hits := &searchHits[H]{
		Hits:   make([]H, len(res.Hits.Hits)),
		Total:  res.Hits.Total.Value,
		Cursor: cursor,
	}
	for i, hit := range res.Hits.Hits {
		var doc map[string]any
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			return nil, fmt.Errorf("opensearchindex: unmarshal hit: %w", err)
		}
		hits.Hits[i] = idx.toHit(hit.ID, doc)
	}

	// Parse facets from aggregations.
	if len(opts.Facets) > 0 && res.Aggregations != nil {
		hits.Facets = idx.parseFacets(opts.Facets, res.Aggregations)
	}

	return hits, nil
}

// buildFilterClauses converts QueryFilter AND conditions to OpenSearch filter clauses.
func (idx *searchIndex[T, H]) buildFilterClauses(conditions []*bbl.AndCondition) ([]any, error) {
	var clauses []any
	for _, cond := range conditions {
		if cond.Terms != nil {
			c, err := idx.buildTermsClause(cond.Terms)
			if err != nil {
				return nil, err
			}
			clauses = append(clauses, c)
		} else if len(cond.Or) > 0 {
			var orClauses []any
			for _, orCond := range cond.Or {
				if orCond.Terms != nil {
					c, err := idx.buildTermsClause(orCond.Terms)
					if err != nil {
						return nil, err
					}
					orClauses = append(orClauses, c)
				} else if len(orCond.And) > 0 {
					andClauses, err := idx.buildFilterClauses(orCond.And)
					if err != nil {
						return nil, err
					}
					orClauses = append(orClauses, boolQuery(must(andClauses...)))
				}
			}
			clauses = append(clauses, boolQuery(should(orClauses...)))
		}
	}
	return clauses, nil
}

func (idx *searchIndex[T, H]) buildTermsClause(f *bbl.TermsFilter) (map[string]any, error) {
	docField, ok := idx.filterDefs[f.Field]
	if !ok {
		return nil, fmt.Errorf("opensearchindex: unknown filter field %q", f.Field)
	}
	return termsQuery(docField, f.Terms), nil
}

// buildFacetAggs builds global aggregations with per-facet filter exclusion.
// Each facet gets the full query as its filter, minus the terms filter for that facet.
func (idx *searchIndex[T, H]) buildFacetAggs(query map[string]any, opts *bbl.SearchOpts) (map[string]any, error) {
	aggs := map[string]any{}

	for _, name := range opts.Facets {
		def, ok := idx.facetDefs[name]
		if !ok {
			return nil, fmt.Errorf("opensearchindex: unknown facet %q", name)
		}

		size := def.Size
		if size == 0 {
			size = 20
		}

		facetAgg := map[string]any{
			"terms": map[string]any{
				"field":         def.Field,
				"size":          size,
				"min_doc_count": 0,
			},
		}

		// Build the filter for this facet: the full query minus this facet's own terms filter.
		facetFilter := idx.buildFacetFilter(query, opts, name)

		aggs[name] = map[string]any{
			"filter": facetFilter,
			"aggs": map[string]any{
				"facet": facetAgg,
			},
		}
	}

	return aggs, nil
}

// buildFacetFilter rebuilds the query with the specified facet's terms filter removed.
func (idx *searchIndex[T, H]) buildFacetFilter(query map[string]any, opts *bbl.SearchOpts, excludeFacet string) map[string]any {
	if opts.Filter == nil {
		return query
	}

	// Deep-copy the query to avoid mutating the original.
	queryBytes, _ := json.Marshal(query)
	var filtered map[string]any
	json.Unmarshal(queryBytes, &filtered)

	// Rebuild filter without the excluded facet's terms filter.
	b, ok := filtered["bool"].(map[string]any)
	if !ok {
		return filtered
	}

	filterSlice, ok := b["filter"].([]any)
	if !ok {
		return filtered
	}

	var newFilter []any
	for _, f := range filterSlice {
		fMap, ok := f.(map[string]any)
		if !ok {
			newFilter = append(newFilter, f)
			continue
		}
		// Check if this is a terms filter for the excluded facet.
		if terms, ok := fMap["terms"].(map[string]any); ok {
			docField := idx.filterDefs[excludeFacet]
			if _, matches := terms[docField]; matches {
				continue // skip this filter
			}
		}
		newFilter = append(newFilter, f)
	}

	if len(newFilter) > 0 {
		b["filter"] = newFilter
	} else {
		delete(b, "filter")
	}

	return filtered
}

// parseFacets extracts facet values from the aggregations response.
func (idx *searchIndex[T, H]) parseFacets(facetNames []string, aggregations json.RawMessage) []bbl.Facet {
	var aggsMap map[string]json.RawMessage
	if err := json.Unmarshal(aggregations, &aggsMap); err != nil {
		return nil
	}

	facetsRaw, ok := aggsMap["facets"]
	if !ok {
		return nil
	}

	var facetsMap map[string]json.RawMessage
	if err := json.Unmarshal(facetsRaw, &facetsMap); err != nil {
		return nil
	}

	var facets []bbl.Facet
	for _, name := range facetNames {
		raw, ok := facetsMap[name]
		if !ok {
			continue
		}

		var facetResp struct {
			Facet struct {
				Buckets []struct {
					Key      string `json:"key"`
					DocCount int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"facet"`
		}
		if err := json.Unmarshal(raw, &facetResp); err != nil {
			continue
		}

		if len(facetResp.Facet.Buckets) == 0 {
			continue
		}

		f := bbl.Facet{Name: name}
		for _, bucket := range facetResp.Facet.Buckets {
			f.Vals = append(f.Vals, bbl.FacetValue{
				Val:   bucket.Key,
				Count: bucket.DocCount,
			})
		}
		facets = append(facets, f)
	}

	return facets
}

const (
	reindexBuffer    = 10 * time.Second
	reindexMaxRounds = 5
)

// reindex creates a new index, bulk indexes all entities, catches up with
// changes that occurred during the bulk phase, then atomically swaps the alias.
func (idx *searchIndex[T, H]) reindex(ctx context.Context, all iter.Seq2[T, error], changed func(since time.Time) iter.Seq2[T, error]) error {
	t0 := time.Now().UTC().Add(-reindexBuffer)

	// Create new timestamped index.
	newIdx := fmt.Sprintf("%s_%s", idx.alias, time.Now().UTC().Format("20060102150405"))
	_, err := idx.client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: newIdx,
		Body:  strings.NewReader(idx.settings),
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: create index %s: %w", newIdx, err)
	}

	// Phase 1: bulk index all entities into the new index.
	bulkIndexer, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client: idx.client,
		Index:  newIdx,
		OnError: func(_ context.Context, err error) {
			if idx.onFail != nil {
				idx.onFail(ctx, "", err)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: create bulk indexer: %w", err)
	}

	for entity, err := range all {
		if err != nil {
			return fmt.Errorf("opensearchindex: bulk: %w", err)
		}
		if err := idx.bulkAdd(ctx, bulkIndexer, entity); err != nil {
			if idx.onFail != nil {
				idx.onFail(ctx, "", err)
			}
		}
	}

	if err := bulkIndexer.Close(ctx); err != nil {
		return fmt.Errorf("opensearchindex: close bulk indexer: %w", err)
	}

	// Phase 2: catch-up loop — index entities changed since t0 into the new index.
	for range reindexMaxRounds {
		n := 0
		for entity, err := range changed(t0) {
			if err != nil {
				return fmt.Errorf("opensearchindex: catch-up: %w", err)
			}
			if err := idx.addToIndex(ctx, newIdx, entity); err != nil {
				return fmt.Errorf("opensearchindex: catch-up index: %w", err)
			}
			n++
		}
		if n == 0 {
			break
		}
		t0 = time.Now().UTC().Add(-reindexBuffer)
	}

	// Phase 3: swap alias to the new index.
	if err := switchAlias(ctx, idx.client, idx.alias, newIdx); err != nil {
		return fmt.Errorf("opensearchindex: switch alias: %w", err)
	}

	// Phase 4: best-effort post-swap sweep.
	// Catches writes that landed between the last catch-up and the alias swap.
	// Errors are reported via onFail, not returned — the reindex is complete.
	for entity, err := range changed(t0) {
		if err != nil {
			if idx.onFail != nil {
				idx.onFail(ctx, "", err)
			}
			break
		}
		if err := idx.addToIndex(ctx, newIdx, entity); err != nil {
			if idx.onFail != nil {
				idx.onFail(ctx, "", err)
			}
		}
	}

	return nil
}

// addToIndex indexes a single entity into a specific index (by name, not alias).
// Used by the catch-up loop during reindex — may re-process entities already
// bulk-indexed at the same version. Version conflicts are expected and ignored.
func (idx *searchIndex[T, H]) addToIndex(ctx context.Context, index string, entity T) error {
	id, version, doc := idx.toDoc(entity)

	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("opensearchindex: marshal doc %s: %w", id, err)
	}

	ver := version
	_, err = idx.client.Index(ctx, opensearchapi.IndexReq{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(b),
		Params: opensearchapi.IndexParams{
			Version:     &ver,
			VersionType: versionType,
		},
	})
	if err != nil && !isVersionConflict(err) {
		return err
	}
	return nil
}

func isVersionConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "version_conflict_engine_exception")
}

// switchAlias atomically points the alias at newIdx and removes it from all other indices.
// Old indices are deleted.
func switchAlias(ctx context.Context, client *opensearchapi.Client, alias, newIdx string) error {
	// Find existing indices for this alias.
	catRes, err := client.Cat.Indices(ctx, &opensearchapi.CatIndicesReq{
		Indices: []string{alias + "_*"},
	})
	if err != nil {
		return err
	}

	var actions []any
	// Add the new index to the alias.
	actions = append(actions, map[string]any{
		"add": map[string]any{"alias": alias, "index": newIdx},
	})
	// Remove old indices from the alias and delete them.
	for _, idx := range catRes.Indices {
		if idx.Index != newIdx {
			actions = append(actions, map[string]any{
				"remove_index": map[string]any{"index": idx.Index},
			})
		}
	}

	body, _ := json.Marshal(map[string]any{"actions": actions})
	_, err = client.Aliases(ctx, opensearchapi.AliasesReq{
		Body: bytes.NewReader(body),
	})
	return err
}

// initAlias ensures the alias exists, creating an initial index if needed.
func initAlias(ctx context.Context, client *opensearchapi.Client, alias string, settings string) error {
	// Check if alias already exists.
	res, err := client.Indices.Alias.Exists(ctx, opensearchapi.AliasExistsReq{
		Alias: []string{alias},
	})
	if res != nil && res.StatusCode == 200 {
		return nil
	}
	if res != nil && res.StatusCode != 404 {
		return err
	}

	// Create initial index with alias.
	idx := alias + "_initial"
	_, err = client.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
		Index: idx,
		Body:  strings.NewReader(settings),
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: create initial index %s: %w", idx, err)
	}

	// Point alias at the new index.
	actionsBody, _ := json.Marshal(map[string]any{
		"actions": []any{
			map[string]any{"add": map[string]any{"alias": alias, "index": idx}},
		},
	})
	_, err = client.Aliases(ctx, opensearchapi.AliasesReq{
		Body: bytes.NewReader(actionsBody),
	})
	if err != nil {
		return fmt.Errorf("opensearchindex: create alias %s: %w", alias, err)
	}

	return nil
}
