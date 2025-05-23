package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"slices"

	"github.com/tidwall/gjson"
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

func unmarshalFilters(b []byte) ([]Filter, error) {
	s := gjson.GetBytes(b, "filters").Array()
	filters := make([]Filter, len(s))
	for i, res := range s {
		var f Filter
		switch t := res.Get("type").String(); t {
		case "and":
			f = &AndClause{}
		case "or":
			f = &OrClause{}
		case "terms":
			f = &TermsFilter{}
		default:
			return nil, fmt.Errorf("unknown filter type %q", t)
		}
		if err := json.Unmarshal([]byte(res.Raw), f); err != nil {
			return nil, err
		}
		filters[i] = f
	}
	return filters, nil
}

type AndClause struct {
	Filters []Filter `json:"filters"`
}

func And(filters ...Filter) *AndClause {
	return &AndClause{Filters: filters}
}

func (*AndClause) isFilter() {}

func (f *AndClause) MarshalJSON() (b []byte, e error) {
	return json.Marshal(struct {
		Type    string   `json:"type"`
		Filters []Filter `json:"filters"`
	}{
		Type:    "and",
		Filters: f.Filters,
	})
}

func (f *AndClause) UnmarshalJSON(b []byte) error {
	filters, err := unmarshalFilters(b)
	if err != nil {
		return err
	}
	f.Filters = filters
	return nil
}

type OrClause struct {
	Filters []Filter `json:"filters"`
}

func Or(filters ...Filter) *OrClause {
	return &OrClause{Filters: filters}
}

func (*OrClause) isFilter() {}

func (f *OrClause) MarshalJSON() (b []byte, e error) {
	return json.Marshal(struct {
		Type    string   `json:"type"`
		Filters []Filter `json:"filters"`
	}{
		Type:    "or",
		Filters: f.Filters,
	})
}

func (f *OrClause) UnmarshalJSON(b []byte) error {
	filters, err := unmarshalFilters(b)
	if err != nil {
		return err
	}
	f.Filters = filters
	return nil
}

type TermsFilter struct {
	Field string   `json:"field"`
	Terms []string `json:"terms"`
}

func Terms(field string, terms ...string) *TermsFilter {
	return &TermsFilter{Field: field, Terms: terms}
}

func (*TermsFilter) isFilter() {}

func (f *TermsFilter) MarshalJSON() (b []byte, e error) {
	return json.Marshal(struct {
		Type  string   `json:"type"`
		Field string   `json:"field"`
		Terms []string `json:"terms"`
	}{
		Type:  "terms",
		Field: f.Field,
		Terms: f.Terms,
	})
}

// TODO make a subfield only containing the query, filter, size (export context etc)?
type SearchOpts struct {
	Query  string     `json:"query,omitempty"`
	Filter *AndClause `json:"filter,omitempty"`
	Size   int        `json:"size"`
	From   int        `json:"from,omitempty"`
	Cursor string     `json:"cursor,omitempty"`
	Facets []string   `json:"facets,omitempty"`
}

func (s *SearchOpts) HasFacetTerm(field, term string) bool {
	if s.Filter == nil {
		return false
	}
	for _, f := range s.Filter.Filters {
		if tf, ok := f.(*TermsFilter); ok {
			if tf.Field == field && slices.Contains(tf.Terms, term) {
				return true
			}
		}
	}
	return false
}

func (s *SearchOpts) AddFilters(filters ...Filter) *SearchOpts {
	if s.Filter == nil {
		s.Filter = &AndClause{Filters: filters}
	} else {
		s.Filter.Filters = append(s.Filter.Filters, filters...)
	}
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
