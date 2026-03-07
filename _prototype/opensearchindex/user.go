package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed user_settings.json
var userSettings string

type userDoc struct {
	Completion []string  `json:"completion"`
	Rec        *bbl.User `json:"rec"`
}

func userToDoc(rec *bbl.User) any {
	doc := userDoc{
		Completion: []string{rec.Username, rec.Name},
		Rec:        rec,
	}
	return &doc
}

func generateUserQuery(q string) (string, error) {
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
