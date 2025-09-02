package bbl

import (
	"context"
	"iter"
)

type Index interface {
	Organizations() RecIndex[*Organization]
	People() RecIndex[*Person]
	Projects() RecIndex[*Project]
	Works() RecIndex[*Work]
}

type RecIndex[T Rec] interface {
	Add(context.Context, T) error
	Get(context.Context, string) (T, error)
	Search(context.Context, *SearchOpts) (*RecHits[T], error)
	NewSwitcher(context.Context) (RecIndexSwitcher[T], error)
}

type RecIndexSwitcher[T Rec] interface {
	Add(context.Context, T) error
	Switch(context.Context) error
}

// TODO make a subfield only containing the query, filter, size (export context etc)?
type SearchOpts struct {
	Query       string       `json:"query,omitempty"`
	QueryFilter *QueryFilter `json:"query_filter,omitempty"`
	Size        int          `json:"size"`
	From        int          `json:"from,omitempty"`
	Cursor      string       `json:"cursor,omitempty"`
	Facets      []string     `json:"facets,omitempty"`
}

func (s *SearchOpts) AddTermsFilter(field string, terms ...string) *SearchOpts {
	if s.QueryFilter == nil {
		s.QueryFilter = &QueryFilter{}
	}
	s.QueryFilter.And = append(s.QueryFilter.And, &AndFilter{Terms: &TermsFilter{
		Field: field,
		Terms: terms,
	}})
	return s
}

type RecHits[T Rec] struct {
	Opts   *SearchOpts `json:"-"`
	Hits   []RecHit[T] `json:"hits"`
	Total  int         `json:"total"`
	Cursor string      `json:"cursor,omitempty"`
	Facets []Facet     `json:"facets,omitempty"`
}

type Facet struct {
	Name string       `json:"name"`
	Vals []FacetValue `json:"vals"`
}

type FacetValue struct {
	Val   string `json:"val"`
	Count int    `json:"count"`
}

type RecHit[T any] struct {
	Rec T `json:"rec"`
}

func SearchIter[T Rec](ctx context.Context, index RecIndex[T], opts *SearchOpts, errPtr *error) iter.Seq[T] {
	o := *opts
	return func(yield func(T) bool) {
		for {
			hits, err := index.Search(ctx, &o)
			if err != nil {
				*errPtr = err
				return
			}
			for _, hit := range hits.Hits {
				if !yield(hit.Rec) {
					return
				}
			}
			if len(hits.Hits) < o.Size {
				return
			}
			o.Cursor = hits.Cursor
		}
	}
}
