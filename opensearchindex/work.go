package opensearchindex

import (
	_ "embed"
	"fmt"

	"github.com/ugent-library/bbl"
)

//go:embed work_settings.json
var workSettings string

type workDoc struct {
	CreatedByID string    `json:"created_by_id,omitempty"`
	PersonID    []string  `json:"person_id,omitempty"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Completion  []string  `json:"completion"`
	Rec         *bbl.Work `json:"rec"`
}

var workTermsFilters = map[string]string{
	"created":     "created_by_id",
	"contributed": "person_id",
	"kind":        "kind",
	"status":      "status",
}

func workToDoc(rec *bbl.Work) any {
	doc := workDoc{
		CreatedByID: rec.CreatedByID,
		Kind:        rec.Kind,
		Status:      rec.Status,
		Rec:         rec,
	}
	for _, con := range rec.Contributors {
		if con.PersonID != "" {
			doc.PersonID = append(doc.PersonID, con.PersonID)
		}
	}
	for _, text := range rec.Attrs.Titles {
		doc.Completion = append(doc.Completion, text.Val)
	}
	return &doc
}

func generateWorkQuery(q string) (string, error) {
	jQ, err := jsonString(q)
	if err != nil {
		return "", err
	}

	j := `{
		"bool": {
			"should": [
				{
					"multi_match": {
						"query": "` + jQ + `",
						"type": "bool_prefix",
						"fields": [
							"completion",
							"completion._2gram",
							"completion._3gram"
						]
					}
				},
				{
					"multi_match": {
						"query": "` + jQ + `",
						"fuzziness": "AUTO",
						"fields": [
							"completion",
							"completion._2gram",
							"completion._3gram"
						]
					}
				}
			]
		}
	}`
	return j, nil
}

func generateWorkAggs(facets []string) (map[string]string, error) {
	m := map[string]string{}
	for _, facet := range facets {
		switch facet {
		case "kind":
			m["kind"] = `{
				"terms": {
					"field": "kind",
					"size": ` + fmt.Sprint(len(bbl.WorkKinds)) + `,
					"min_doc_count": 0
				}
			}`
		case "status":
			m["status"] = `{
				"terms": {
					"field": "status",
					"size": ` + fmt.Sprint(len(bbl.WorkStatuses)) + `,
					"min_doc_count": 0
				}
			}`
		default:
			return nil, fmt.Errorf("unknown facet %s", facet)
		}
	}
	return m, nil
}
