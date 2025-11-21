package bbl

import (
	"slices"

	participle "github.com/alecthomas/participle/v2"
)

type QueryFilter struct {
	And []*AndFilter `parser:"@@ ( 'and' @@ )*" json:"and,omitempty"`
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
	Or    []*OrFilter  `parser:"'(' @@ ( 'or' @@ )+')'" json:"or,omitempty"`
	Terms *TermsFilter `parser:"| @@" json:"terms,omitempty"`
}

type OrFilter struct {
	And   []*AndFilter `parser:"'(' @@ ( 'and' @@ )+')'" json:"and,omitempty"`
	Terms *TermsFilter `parser:"| @@" json:"terms,omitempty"`
}

type TermsFilter struct {
	Field string   `parser:"@Ident '='" json:"field"`
	Terms []string `parser:"@String ( '|' @String )*" json:"terms"`
}

var queryFilterParser = participle.MustBuild[QueryFilter](
	participle.Unquote("String"),
)

func ParseQueryFilter(str string) (*QueryFilter, error) {
	qf, err := queryFilterParser.ParseString("", str)
	if err != nil {
		return nil, err
	}

	return qf, nil
}
