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
	Search(context.Context, SearchOpts) (*RecHits[T], error)
	NewSwitcher(context.Context) (RecIndexSwitcher[T], error)
}

type RecIndexSwitcher[T Rec] interface {
	Add(context.Context, T) error
	Switch(context.Context) error
}

type SearchOpts struct {
	Query   string   `json:"query,omitempty"`
	Filters Filters  `json:"filters,omitempty"`
	Size    int      `json:"size"`
	From    int      `json:"from,omitempty"`
	Cursor  string   `json:"cursor,omitempty"`
	Facets  []string `json:"facets,omitempty"`
}

type RecHits[T Rec] struct {
	Hits    []RecHit[T] `json:"hits"`
	Total   int         `json:"total"`
	Query   string      `json:"query,omitempty"`
	Filters Filters     `json:"filters,omitempty"`
	Size    int         `json:"size"`
	From    int         `json:"from,omitempty"`
	Cursor  string      `json:"cursor,omitempty"`
	Facets  []Facet     `json:"facets,omitempty"`
}

type Filters map[string][]string

func (f Filters) HasVal(k, v string) bool {
	if vals, ok := f[k]; ok {
		return slices.Contains(vals, v)
	}
	return false
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
