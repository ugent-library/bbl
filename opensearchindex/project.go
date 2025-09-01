package opensearchindex

import (
	_ "embed"

	"github.com/ugent-library/bbl"
)

//go:embed project_settings.json
var projectSettings string

type projectDoc struct {
	Completion []string     `json:"completion"`
	Rec        *bbl.Project `json:"rec"`
}

func projectToDoc(rec *bbl.Project) any {
	doc := projectDoc{
		Completion: make([]string, len(rec.Names)),
		Rec:        rec,
	}
	for i, text := range rec.Names {
		doc.Completion[i] = text.Val
	}
	return &doc
}

func generateProjectQuery(q string) (string, error) {
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
