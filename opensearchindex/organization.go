package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed organization_settings.json
var organizationSettings string

var organizationFilterDefs = map[string]string{
	"kind": "kind",
}

var organizationFacetDefs = map[string]facetDef{
	"kind": {Field: "kind", Size: 50},
}

func organizationToDoc(o *bbl.Organization) (id string, version int, doc map[string]any) {
	name := ""
	var completion []string
	for _, n := range o.Names {
		if name == "" {
			name = n.Val
		}
		completion = append(completion, n.Val)
	}

	idStr := o.ID.String()
	return idStr, o.Version, map[string]any{
		"id":         idStr,
		"kind":       o.Kind,
		"name":       name,
		"completion": completion,
	}
}

func organizationToHit(id string, doc map[string]any) bbl.OrganizationHit {
	hit := bbl.OrganizationHit{}
	hit.ID.UnmarshalText([]byte(id))

	if v, ok := doc["kind"].(string); ok {
		hit.Kind = v
	}
	if v, ok := doc["name"].(string); ok {
		hit.Name = v
	}
	return hit
}

func buildOrganizationQuery(q string) map[string]any {
	completionFields := []string{"completion", "completion._2gram", "completion._3gram"}
	return boolQuery(
		should(
			multiMatch(q, completionFields, "bool_prefix"),
			fuzzyMultiMatch(q, completionFields),
		),
		minimumShouldMatch(1),
	)
}
