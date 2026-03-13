package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed project_settings.json
var projectSettings string

var projectFilterDefs = map[string]string{
	"status": "status",
}

var projectFacetDefs = map[string]facetDef{
	"status": {Field: "status", Size: 10},
}

func projectToDoc(p *bbl.Project) (id string, version int, doc map[string]any) {
	idStr := p.ID.String()
	var title string
	var completion []string
	for _, t := range p.Titles {
		if title == "" {
			title = t.Val
		}
		completion = append(completion, t.Val)
	}
	return idStr, p.Version, map[string]any{
		"id":         idStr,
		"status":     p.Status,
		"title":      title,
		"completion": completion,
	}
}

func projectToHit(id string, doc map[string]any) bbl.ProjectHit {
	hit := bbl.ProjectHit{}
	hit.ID.UnmarshalText([]byte(id))

	if v, ok := doc["status"].(string); ok {
		hit.Status = v
	}
	if v, ok := doc["title"].(string); ok {
		hit.Title = v
	}
	return hit
}

func buildProjectQuery(q string) map[string]any {
	completionFields := []string{"completion", "completion._2gram", "completion._3gram"}
	return boolQuery(
		should(
			multiMatch(q, completionFields, "bool_prefix"),
			fuzzyMultiMatch(q, completionFields),
		),
		minimumShouldMatch(1),
	)
}
