package bbl

import (
	"slices"

	participle "github.com/alecthomas/participle/v2"
)

// QueryFilter represents a conjunction of conditions for search filtering.
//
// Filter expressions select documents by field values using a simple query language:
//
//	field=value             exact match
//	field=val1|val2         match any of the values
//	field=a field=b         AND (both must match)
//	field=a or field=b      OR (either must match)
//	(field=a or field=b)    grouping with parentheses
//
// Examples:
//
//	status=public
//	status=public kind=book|article
//	kind=book or kind=conference_paper
//	status=public (kind=book or kind=article)
//	(status=public and kind=book) or (status=private and kind=article)
//
// Use [ParseQueryFilter] to parse a filter expression string into a QueryFilter.
type QueryFilter struct {
	And []*AndCondition `json:"and"`
}

// HasTerm returns true if the filter contains a terms filter for the given field and term.
func (qf *QueryFilter) HasTerm(field, term string) bool {
	if qf == nil {
		return false
	}
	for _, f := range qf.And {
		if f.Terms != nil && f.Terms.Field == field && slices.Contains(f.Terms.Terms, term) {
			return true
		}
	}
	return false
}

// AndCondition is a single clause in a conjunction. It is either an OR group or a terms filter.
type AndCondition struct {
	Or    []*OrCondition `json:"or,omitempty"`
	Terms *TermsFilter   `json:"terms,omitempty"`
}

// OrCondition is a single clause in a disjunction. It is either an AND group or a terms filter.
type OrCondition struct {
	And   []*AndCondition `json:"and,omitempty"`
	Terms *TermsFilter    `json:"terms,omitempty"`
}

// TermsFilter matches documents where the field contains any of the given terms.
type TermsFilter struct {
	Field string   `json:"field"`
	Terms []string `json:"terms"`
}

// Parser grammar types (internal).
type grammar struct {
	Or []*orCondition `parser:"@@ ( ('or' | 'OR') @@ )*"`
}

type orCondition struct {
	And []*andCondition `parser:"@@ ( ('and' | 'AND') @@ )*"`
}

type andCondition struct {
	Or     []*orCondition `parser:"'(' @@ ( ('or' | 'OR') @@ )* ')'"`
	Filter *filter        `parser:"| @@"`
}

type filter struct {
	Field string   `parser:"@Ident"`
	Op    string   `parser:"@( '>=' | '>' | '<=' | '<' | '=' )"`
	Terms []string `parser:"( @Ident | @String ) ( '|' ( @Ident | @String ) )*"`
}

var queryParser = participle.MustBuild[grammar](
	participle.Unquote("String"),
)

// ParseQueryFilter parses a search filter expression like "status=public kind=book|article".
func ParseQueryFilter(str string) (*QueryFilter, error) {
	g, err := queryParser.ParseString("", str)
	if err != nil {
		return nil, err
	}

	cond := &AndCondition{}
	for _, c := range g.Or {
		cond.Or = append(cond.Or, visitOrCondition(c))
	}
	if len(cond.Or) == 1 && cond.Or[0].Terms != nil {
		cond.Terms = cond.Or[0].Terms
		cond.Or = nil
	}

	qf := &QueryFilter{}
	if len(cond.Or) == 1 {
		if cond.Or[0].Terms != nil {
			qf.And = []*AndCondition{{Terms: cond.Or[0].Terms}}
		} else {
			qf.And = cond.Or[0].And
		}
	} else {
		qf.And = []*AndCondition{cond}
	}

	return qf, nil
}

func visitAndCondition(o *andCondition) *AndCondition {
	cond := &AndCondition{}

	if o.Filter != nil {
		cond.Terms = &TermsFilter{Field: o.Filter.Field, Terms: o.Filter.Terms}
	} else {
		for _, c := range o.Or {
			cond.Or = append(cond.Or, visitOrCondition(c))
		}
	}

	if len(cond.Or) == 1 {
		if cond.Or[0].Terms != nil {
			cond.Terms = cond.Or[0].Terms
			cond.Or = nil
		} else if len(cond.Or[0].And) == 1 {
			cond.Or = cond.Or[0].And[0].Or
		}
	}

	return cond
}

func visitOrCondition(o *orCondition) *OrCondition {
	cond := &OrCondition{}

	for _, c := range o.And {
		cond.And = append(cond.And, visitAndCondition(c))
	}

	if len(cond.And) == 1 {
		if cond.And[0].Terms != nil {
			cond.Terms = cond.And[0].Terms
			cond.And = nil
		} else if len(cond.And[0].Or) == 1 {
			cond.And = cond.And[0].Or[0].And
		}
	}

	return cond
}
