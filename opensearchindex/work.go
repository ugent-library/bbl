package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed work_settings.json
var workSettings string

var workFilterDefs = map[string]string{
	"kind":        "kind",
	"status":      "status",
	"contributor": "person_ids",
}

var workFacetDefs = map[string]facetDef{
	"kind":   {Field: "kind", Size: 50},
	"status": {Field: "status", Size: 10},
}

func workToDoc(w *bbl.Work) (id string, version int, doc map[string]any) {
	title := ""
	var completion []string
	for _, t := range w.Titles {
		if title == "" {
			title = t.Val
		}
		completion = append(completion, t.Val)
	}

	var identifiers []string
	for _, ident := range w.Identifiers {
		identifiers = append(identifiers, ident.Scheme+":"+ident.Val)
	}

	var personIDs []string
	for _, c := range w.Contributors {
		if c.PersonID != nil {
			personIDs = append(personIDs, c.PersonID.String())
		}
	}

	idStr := w.ID.String()
	return idStr, w.Version, map[string]any{
		"id":          idStr,
		"kind":        w.Kind,
		"status":      w.Status,
		"title":       title,
		"identifiers": identifiers,
		"person_ids":  personIDs,
		"completion":  completion,
	}
}

func workToHit(id string, doc map[string]any) bbl.WorkHit {
	hit := bbl.WorkHit{}
	hit.ID.UnmarshalText([]byte(id))

	if v, ok := doc["kind"].(string); ok {
		hit.Kind = v
	}
	if v, ok := doc["status"].(string); ok {
		hit.Status = v
	}
	if v, ok := doc["title"].(string); ok {
		hit.Title = v
	}
	return hit
}

func buildWorkQuery(q string) map[string]any {
	completionFields := []string{"completion", "completion._2gram", "completion._3gram"}
	return boolQuery(
		should(
			termQuery("identifiers", q),
			multiMatch(q, completionFields, "bool_prefix"),
			fuzzyMultiMatch(q, completionFields),
		),
		minimumShouldMatch(1),
	)
}
