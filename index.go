package bbl

import (
	"context"
	"iter"
	"time"
)

// Facet represents a faceted search result with counts per value.
type Facet struct {
	Name string       `json:"name"`
	Vals []FacetValue `json:"vals"`
}

// FacetValue is a single value in a facet with its document count.
type FacetValue struct {
	Val   string `json:"val"`
	Count int    `json:"count"`
}

// SearchOpts controls a search query.
type SearchOpts struct {
	Query  string       `json:"query,omitempty"`
	Filter *QueryFilter `json:"filter,omitempty"`
	Facets []string     `json:"facets,omitempty"`
	Size   int          `json:"size"`
	Cursor string       `json:"cursor,omitempty"` // base64-encoded search_after; mutually exclusive with Offset
	Offset int          `json:"offset,omitempty"` // for UI pagination; hard max enforced by implementation
}

// WorkHit is a search result for a work, containing display fields.
type WorkHit struct {
	ID     ID     `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

// WorkHits is the result of a work search query.
type WorkHits struct {
	Hits   []WorkHit `json:"hits"`
	Total  int       `json:"total"`
	Cursor string    `json:"cursor,omitempty"`
	Facets []Facet   `json:"facets,omitempty"`
}

// WorkRecordHit is a search result containing the full work record.
type WorkRecordHit struct {
	Work *Work `json:"work"`
}

// WorkRecordHits is the result of a work search with full records.
type WorkRecordHits struct {
	Hits   []WorkRecordHit `json:"hits"`
	Total  int             `json:"total"`
	Cursor string          `json:"cursor,omitempty"`
	Facets []Facet         `json:"facets,omitempty"`
}

// PersonHit is a search result for a person, containing display fields.
type PersonHit struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

// PersonHits is the result of a person search query.
type PersonHits struct {
	Hits   []PersonHit `json:"hits"`
	Total  int         `json:"total"`
	Cursor string      `json:"cursor,omitempty"`
	Facets []Facet     `json:"facets,omitempty"`
}

// ProjectHit is a search result for a project, containing display fields.
type ProjectHit struct {
	ID     ID     `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

// ProjectHits is the result of a project search query.
type ProjectHits struct {
	Hits   []ProjectHit `json:"hits"`
	Total  int          `json:"total"`
	Cursor string       `json:"cursor,omitempty"`
	Facets []Facet      `json:"facets,omitempty"`
}

// OrganizationHit is a search result for an organization, containing display fields.
type OrganizationHit struct {
	ID   ID     `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// OrganizationHits is the result of an organization search query.
type OrganizationHits struct {
	Hits   []OrganizationHit `json:"hits"`
	Total  int               `json:"total"`
	Cursor string            `json:"cursor,omitempty"`
	Facets []Facet           `json:"facets,omitempty"`
}

// WorkIndex is the search index for works.
type WorkIndex interface {
	Add(ctx context.Context, work *Work) error
	Delete(ctx context.Context, id ID) error
	DeleteAll(ctx context.Context) error
	Search(ctx context.Context, opts *SearchOpts) (*WorkHits, error)
	Reindex(ctx context.Context, all iter.Seq2[*Work, error], changed func(since time.Time) iter.Seq2[*Work, error]) error
}

// PersonIndex is the search index for people.
type PersonIndex interface {
	Add(ctx context.Context, person *Person) error
	Delete(ctx context.Context, id ID) error
	DeleteAll(ctx context.Context) error
	Search(ctx context.Context, opts *SearchOpts) (*PersonHits, error)
	Reindex(ctx context.Context, all iter.Seq2[*Person, error], changed func(since time.Time) iter.Seq2[*Person, error]) error
}

// ProjectIndex is the search index for projects.
type ProjectIndex interface {
	Add(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id ID) error
	DeleteAll(ctx context.Context) error
	Search(ctx context.Context, opts *SearchOpts) (*ProjectHits, error)
	Reindex(ctx context.Context, all iter.Seq2[*Project, error], changed func(since time.Time) iter.Seq2[*Project, error]) error
}

// OrganizationIndex is the search index for organizations.
type OrganizationIndex interface {
	Add(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id ID) error
	DeleteAll(ctx context.Context) error
	Search(ctx context.Context, opts *SearchOpts) (*OrganizationHits, error)
	Reindex(ctx context.Context, all iter.Seq2[*Organization, error], changed func(since time.Time) iter.Seq2[*Organization, error]) error
}

// Index provides access to per-entity search indexes.
type Index interface {
	Works() WorkIndex
	People() PersonIndex
	Projects() ProjectIndex
	Organizations() OrganizationIndex
}

const searchAllSize = 1000

// SearchAllWorks returns an iterator over all work hits matching the query,
// using cursor-based pagination internally.
func SearchAllWorks(ctx context.Context, idx WorkIndex, opts *SearchOpts) iter.Seq2[WorkHit, error] {
	return searchAll(ctx, opts, func(ctx context.Context, o *SearchOpts) ([]WorkHit, string, error) {
		res, err := idx.Search(ctx, o)
		if err != nil {
			return nil, "", err
		}
		return res.Hits, res.Cursor, nil
	})
}

// SearchAllPeople returns an iterator over all person hits matching the query.
func SearchAllPeople(ctx context.Context, idx PersonIndex, opts *SearchOpts) iter.Seq2[PersonHit, error] {
	return searchAll(ctx, opts, func(ctx context.Context, o *SearchOpts) ([]PersonHit, string, error) {
		res, err := idx.Search(ctx, o)
		if err != nil {
			return nil, "", err
		}
		return res.Hits, res.Cursor, nil
	})
}

// SearchAllProjects returns an iterator over all project hits matching the query.
func SearchAllProjects(ctx context.Context, idx ProjectIndex, opts *SearchOpts) iter.Seq2[ProjectHit, error] {
	return searchAll(ctx, opts, func(ctx context.Context, o *SearchOpts) ([]ProjectHit, string, error) {
		res, err := idx.Search(ctx, o)
		if err != nil {
			return nil, "", err
		}
		return res.Hits, res.Cursor, nil
	})
}

// SearchAllOrganizations returns an iterator over all organization hits matching the query.
func SearchAllOrganizations(ctx context.Context, idx OrganizationIndex, opts *SearchOpts) iter.Seq2[OrganizationHit, error] {
	return searchAll(ctx, opts, func(ctx context.Context, o *SearchOpts) ([]OrganizationHit, string, error) {
		res, err := idx.Search(ctx, o)
		if err != nil {
			return nil, "", err
		}
		return res.Hits, res.Cursor, nil
	})
}

// searchAll is the generic cursor-tailing iterator. It pages through results
// using search_after cursors until no more hits are returned.
func searchAll[H any](ctx context.Context, opts *SearchOpts, search func(context.Context, *SearchOpts) ([]H, string, error)) iter.Seq2[H, error] {
	return func(yield func(H, error) bool) {
		o := &SearchOpts{
			Query:  opts.Query,
			Filter: opts.Filter,
			Size:   searchAllSize,
		}
		for {
			hits, cursor, err := search(ctx, o)
			if err != nil {
				var zero H
				yield(zero, err)
				return
			}
			for _, h := range hits {
				if !yield(h, nil) {
					return
				}
			}
			if cursor == "" || len(hits) < o.Size {
				return
			}
			o.Cursor = cursor
		}
	}
}
