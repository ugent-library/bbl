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
	Query   string   `json:"query,omitempty"`
	Filters []Filter `json:"filters,omitempty"`
	Size    int      `json:"size"`
	From    int      `json:"from,omitempty"`
	Cursor  string   `json:"cursor,omitempty"`
	Facets  []string `json:"facets,omitempty"`
}

type Filter interface {
	isFilter()
}

type AndClause struct {
	Filters []Filter
}

func (*AndClause) isFilter() {}

type OrClause struct {
	Filters []Filter
}

func (*OrClause) isFilter() {}

type TermsFilter struct {
	Field string
	Terms []string
}

func (*TermsFilter) isFilter() {}

func (s *SearchOpts) HasFilterTerm(field, term string) bool {
	for _, f := range s.Filters {
		if tf, ok := f.(*TermsFilter); ok {
			if tf.Field == field && slices.Contains(tf.Terms, term) {
				return true
			}
		}
	}
	return false
}

func (s *SearchOpts) SetTermsFilter(field string, terms ...string) *SearchOpts {
	for _, f := range s.Filters {
		if tf, ok := f.(*TermsFilter); ok {
			if tf.Field == field {
				tf.Terms = terms
				return s
			}
		}
	}
	tf := &TermsFilter{Field: field, Terms: terms}
	s.Filters = append(s.Filters, tf)
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
