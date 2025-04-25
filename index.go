package bbl

import (
	"context"
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
	Query  string   `json:"query,omitempty"`
	Size   int      `json:"size"`
	From   int      `json:"from,omitempty"`
	Cursor string   `json:"cursor,omitempty"`
	Facets []string `json:"facets,omitempty"`
}

type RecHits[T Rec] struct {
	Hits   []RecHit[T] `json:"hits"`
	Total  int         `json:"total"`
	Query  string      `json:"query,omitempty"`
	Size   int         `json:"size"`
	From   int         `json:"from,omitempty"`
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
