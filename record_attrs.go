package bbl

type RecordIdentifiers struct {
	Identifiers []Attr[Code] `json:"identifiers,omitempty"`
}

func (rec RecordIdentifiers) IdentifiersWithScheme(scheme string) []string {
	var codes []string
	for _, attr := range rec.Identifiers {
		if attr.Val.Scheme == scheme {
			codes = append(codes, attr.Val.Code)
		}
	}
	return codes
}

func (rec RecordIdentifiers) IdentifierWithScheme(scheme string) string {
	for _, attr := range rec.Identifiers {
		if attr.Val.Scheme == scheme {
			return attr.Val.Code
		}
	}
	return ""
}
