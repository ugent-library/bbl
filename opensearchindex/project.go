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
		Completion: make([]string, len(rec.Attrs.Names)),
		Rec:        rec,
	}
	for i, text := range rec.Attrs.Names {
		doc.Completion[i] = text.Val
	}
	return &doc
}

func generateProjectQuery(str string) (string, error) {
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
