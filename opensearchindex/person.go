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
		Completion: []string{rec.Attrs.Name},
		Rec:        rec,
	}
	return &doc
}

func generatePersonQuery(str string) (string, error) {
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
							"record.fields.name.suggest",
							"record.fields.name.suggest._2gram",
							"record.fields.name.suggest._3gram"
						]
					}
				},
				{
					"multi_match": {
						"query": "` + jsonStr + `",
						"fuzziness": "AUTO",
						"fields": [
							"record.fields.name.suggest",
							"record.fields.name.suggest._2gram",
							"record.fields.name.suggest._3gram"
						]
					}
				}
			]
		}
	}`
	return q, nil
}
