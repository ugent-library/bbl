package opensearchindex

// boolQuery builds an OpenSearch bool query using functional options.
func boolQuery(opts ...func(map[string]any)) map[string]any {
	b := map[string]any{}
	for _, opt := range opts {
		opt(b)
	}
	return map[string]any{"bool": b}
}

// must adds a must clause to a bool query.
func must(clauses ...any) func(map[string]any) {
	return func(b map[string]any) {
		b["must"] = clauses
	}
}

// should adds a should clause to a bool query.
func should(clauses ...any) func(map[string]any) {
	return func(b map[string]any) {
		b["should"] = clauses
	}
}

// filter adds a filter clause to a bool query.
func filter(clauses ...any) func(map[string]any) {
	return func(b map[string]any) {
		b["filter"] = clauses
	}
}

// minimumShouldMatch sets the minimum_should_match parameter on a bool query.
func minimumShouldMatch(n int) func(map[string]any) {
	return func(b map[string]any) {
		b["minimum_should_match"] = n
	}
}

// termsQuery builds a terms query for exact matching.
func termsQuery(field string, vals []string) map[string]any {
	return map[string]any{
		"terms": map[string]any{
			field: vals,
		},
	}
}

// termQuery builds a term query for a single exact value.
func termQuery(field string, val string) map[string]any {
	return map[string]any{
		"term": map[string]any{
			field: val,
		},
	}
}

// multiMatch builds a multi_match query across multiple fields.
// matchType can be "bool_prefix", "phrase", or empty for default best_fields.
func multiMatch(query string, fields []string, matchType string) map[string]any {
	m := map[string]any{
		"query":  query,
		"fields": fields,
	}
	if matchType != "" {
		m["type"] = matchType
	}
	return map[string]any{"multi_match": m}
}

// fuzzyMultiMatch builds a multi_match query with fuzziness.
func fuzzyMultiMatch(query string, fields []string) map[string]any {
	return map[string]any{
		"multi_match": map[string]any{
			"query":     query,
			"fields":    fields,
			"fuzziness": "AUTO",
		},
	}
}

// matchAll builds a match_all query.
func matchAll() map[string]any {
	return map[string]any{"match_all": map[string]any{}}
}
