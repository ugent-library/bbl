package opensearchindex

import "encoding/json"

// func encodeCursor(res *opensearchapi.SearchResp, args biblio.SearchArgs) (string, error) {
// 	n := len(res.Hits.Hits)
// 	if n == 0 || n < args.Limit {
// 		return "", nil
// 	}
// 	c, err := json.Marshal(res.Hits.Hits[n-1].Sort)
// 	if err != nil {
// 		return "", err
// 	}
// 	return base64.StdEncoding.EncodeToString(c), nil
// }

func jsonString(str string) (string, error) {
	b, err := json.Marshal(str)
	if err != nil {
		return "", err
	}
	s := string(b)
	return s[1 : len(s)-1], nil
}
