package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed person_settings.json
var personSettings string

var personFilterDefs = map[string]string{}

var personFacetDefs = map[string]facetDef{}

func personToDoc(p *bbl.Person) (id string, version int, doc map[string]any) {
	name := p.Name
	if name == "" {
		parts := []string{}
		if p.GivenName != "" {
			parts = append(parts, p.GivenName)
		}
		if p.FamilyName != "" {
			parts = append(parts, p.FamilyName)
		}
		for i, part := range parts {
			if i > 0 {
				name += " "
			}
			name += part
		}
	}

	var completion []string
	if name != "" {
		completion = append(completion, name)
	}

	idStr := p.ID.String()
	return idStr, p.Version, map[string]any{
		"id":         idStr,
		"name":       name,
		"completion": completion,
	}
}

func personToHit(id string, doc map[string]any) bbl.PersonHit {
	hit := bbl.PersonHit{}
	hit.ID.UnmarshalText([]byte(id))

	if v, ok := doc["name"].(string); ok {
		hit.Name = v
	}
	return hit
}

func buildPersonQuery(q string) map[string]any {
	completionFields := []string{"completion", "completion._2gram", "completion._3gram"}
	return boolQuery(
		should(
			multiMatch(q, completionFields, "bool_prefix"),
			fuzzyMultiMatch(q, completionFields),
		),
		minimumShouldMatch(1),
	)
}
