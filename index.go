package bbl

import (
	"context"
	"slices"
)

type Index interface {
	Organizations() RecIndex[*Organization]
	People() RecIndex[*Person]
	Projects() RecIndex[*Project]
	Works() RecIndex[*Work]
}

type RecIndex[T Rec] interface {
	Add(context.Context, T) error
	Search(context.Context, *SearchOpts) (*RecHits[T], error)
	NewSwitcher(context.Context) (RecIndexSwitcher[T], error)
}

type RecIndexSwitcher[T Rec] interface {
	Add(context.Context, T) error
	Switch(context.Context) error
}

type SearchOpts struct {
	Query   string              `json:"query,omitempty"`
	Filters map[string][]string `json:"filters,omitempty"`
	Size    int                 `json:"size"`
	From    int                 `json:"from,omitempty"`
	Cursor  string              `json:"cursor,omitempty"`
	Facets  []string            `json:"facets,omitempty"`
}

func (s *SearchOpts) HasFilterVal(k, v string) bool {
	if vals, ok := s.Filters[k]; ok {
		return slices.Contains(vals, v)
	}
	return false
}

func (s *SearchOpts) SetFilterVal(k, v string) *SearchOpts {
	if s.Filters == nil {
		s.Filters = make(map[string][]string)
	}
	s.Filters[k] = []string{v}
	return s
}

type RecHits[T Rec] struct {
	Opts   *SearchOpts
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
