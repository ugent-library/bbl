package opensearchindex

import (
	_ "embed"
	"fmt"

	"github.com/tidwall/sjson"
	"github.com/ugent-library/bbl"
)

//go:embed work_settings.json
var workSettings string

type workDoc struct {
	Completion []string  `json:"completion"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	Rec        *bbl.Work `json:"rec"`
}

func workToDoc(rec *bbl.Work) any {
	doc := workDoc{
		Completion: make([]string, len(rec.Attrs.Titles)),
		Kind:       rec.Kind,
		Status:     rec.Status,
		Rec:        rec,
	}
	for i, text := range rec.Attrs.Titles {
		doc.Completion[i] = text.Val
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

func generateWorkFilters(filters map[string][]string) (map[string]string, error) {
	m := map[string]string{}
	for filter, vals := range filters {
		switch filter {
		case "kind":
			f, err := sjson.Set(``, "terms.kind", vals)
			if err != nil {
				return nil, err
			}
			m[filter] = f
		case "status":
			f, err := sjson.Set(``, "terms.kind", vals)
			if err != nil {
				return nil, err
			}
			m[filter] = f
		default:
			return nil, fmt.Errorf("unknown filter %s", filter)
		}

	}
	return m, nil
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
