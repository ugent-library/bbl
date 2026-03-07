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
		Completion: make([]string, len(rec.Names)),
		Rec:        rec,
	}
	for i, text := range rec.Names {
		doc.Completion[i] = text.Val
	}
	for _, iden := range rec.Identifiers {
		doc.Completion = append(doc.Completion, iden.Val)
	}
	return &doc
}

func generateOrganizationQuery(q string) (string, error) {
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
