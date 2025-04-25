package opensearchindex

import (
	_ "embed"

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

func generateWorkQuery(str string) (string, error) {
	jsonStr, err := jsonString(str)
	if err != nil {
		return "", err
	}
	q := `{
		"bool": {
			"should": [
				{
					"multi_match": {
						"query": "` + jsonStr + `",
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
						"query": "` + jsonStr + `",
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
	return q, nil
}
