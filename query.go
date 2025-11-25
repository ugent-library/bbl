package bbl

import (
	"encoding/json"
	"log"
	"slices"

	participle "github.com/alecthomas/participle/v2"
)

type QueryFilter struct {
	And []*AndFilter `json:"and"`
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

type AndFilter struct {
	Or    []*OrFilter  `parser:"'(' @@ ( 'or' @@ )+ ')'" json:"or,omitempty"`
	Terms *TermsFilter `parser:"| @@" json:"terms,omitempty"`
}

type OrFilter struct {
	And   []*AndFilter `parser:"'(' @@ ( 'and' @@ )+ ')'" json:"and,omitempty"`
	Terms *TermsFilter `parser:"| @@" json:"terms,omitempty"`
}

type TermsFilter struct {
	Field string   `parser:"@Ident '='" json:"field"`
	Terms []string `parser:"@String ( '|' @String )*" json:"terms"`
}

type andCondition struct {
	Or []*orCondition `parser:"@@ ( 'or' @@ )*"`
}

type orCondition struct {
	And []*expression `parser:"@@ ( 'and' @@ )*"`
}

type expression struct {
	Or     []*orCondition `parser:"'(' @@ ( 'or' @@ )* ')'"`
	Filter *filter        `parser:"| @@"`
}

type filter struct {
	Field string   `parser:"@Ident" json:"field"`
	Op    string   `parser:"@( '>=' | '>' | '<=' | '<' | '=' )" json:"op"`
	Terms []string `parser:"( @Ident | @String ) ( '|' ( @Ident | @String ) )*" json:"terms"`
}

var queryParser = participle.MustBuild[andCondition](
	participle.Unquote("String"),
)

func ParseQueryFilter(str string) (*QueryFilter, error) {
	g, err := queryParser.ParseString("", str)
	if err != nil {
		return nil, err
	}

	j, _ := json.MarshalIndent(g, "", "  ")
	log.Printf("filter: %s", j)

	// qf := &QueryFilter{}

	// if g.And != nil {
	// 	qf.And = append([]*AndFilter{{Terms: g.Terms}}, g.And...)
	// } else if g.Or != nil {
	// 	qf.And = []*AndFilter{{Or: append([]*OrFilter{{Terms: g.Terms}}, g.Or...)}}
	// } else {
	// 	qf.And = []*AndFilter{{Terms: g.Terms}}
	// }

	// return qf, nil

	return &QueryFilter{}, nil
}
