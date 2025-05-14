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

type Filter interface {
	isFilter()
}

type AndClause struct {
	Filters []Filter
}

func (*AndClause) isFilter() {}

func And(filters ...Filter) *AndClause {
	return &AndClause{Filters: filters}
}

type OrClause struct {
	Filters []Filter
}

func (*OrClause) isFilter() {}

func Or(filters ...Filter) *OrClause {
	return &OrClause{Filters: filters}
}

type TermsFilter struct {
	Field string
	Terms []string
}

func (*TermsFilter) isFilter() {}

func Terms(field string, terms ...string) *TermsFilter {
	return &TermsFilter{Field: field, Terms: terms}
}

type SearchOpts struct {
	Query   string   `json:"query,omitempty"`
	Filters []Filter `json:"filters,omitempty"`
	Size    int      `json:"size"`
	From    int      `json:"from,omitempty"`
	Cursor  string   `json:"cursor,omitempty"`
	Facets  []string `json:"facets,omitempty"`
}

func (s *SearchOpts) HasFacetTerm(field, term string) bool {
	for _, f := range s.Filters {
		if tf, ok := f.(*TermsFilter); ok {
			if tf.Field == field && slices.Contains(tf.Terms, term) {
				return true
			}
		}
	}
	return false
}

func (s *SearchOpts) AddFilters(filters ...Filter) *SearchOpts {
	s.Filters = append(s.Filters, filters...)
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
