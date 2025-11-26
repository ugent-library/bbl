package bbl

import (
	"encoding/json"
	"log"
	"slices"

	participle "github.com/alecthomas/participle/v2"
)

type QueryFilter struct {
	And []*AndCondition `json:"and"`
}

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

type AndCondition struct {
	Or    []*OrCondition `json:"or,omitempty"`
	Terms *TermsFilter   `json:"terms,omitempty"`
}

type OrCondition struct {
	And   []*AndCondition `json:"and,omitempty"`
	Terms *TermsFilter    `json:"terms,omitempty"`
}

type TermsFilter struct {
	Field string   `json:"field"`
	Terms []string `json:"terms"`
}

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

// TODO handle >=, >, <=, < operators
// TODO add negation
var queryParser = participle.MustBuild[grammar](
	participle.Unquote("String"),
)

// TODO optimize "status=draft or kind=book or status=public" to "status=draft|public or kind=book"
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

	// TODO remove this
	j, _ := json.MarshalIndent(qf, "", "  ")
	log.Printf("parsed queryfilter: %s", j)

	return qf, nil
}
