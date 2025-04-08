package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed organization_settings.json
var organizationSettings string

type organizationDoc struct {
	Completion []string          `json:"completion"`
	Rec        *bbl.Organization `json:"rec"`
}

func organizationToDoc(rec *bbl.Organization) any {
	doc := organizationDoc{
		Completion: make([]string, len(rec.Attrs.Names)),
		Rec:        rec,
	}
	for i, text := range rec.Attrs.Names {
		doc.Completion[i] = text.Val
	}
	return &doc
}

func generateOrganizationQuery(str string) (string, error) {
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
