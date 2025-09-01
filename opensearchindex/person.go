package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed person_settings.json
var personSettings string

type personDoc struct {
	Completion []string    `json:"completion"`
	Rec        *bbl.Person `json:"rec"`
}

func personToDoc(rec *bbl.Person) any {
	doc := personDoc{
		Completion: []string{rec.Name},
		Rec:        rec,
	}
	return &doc
}

func generatePersonQuery(q string) (string, error) {
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
