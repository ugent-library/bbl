package bbl

var projectKind = "project"

var projectSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"abstract":   {},
		"identifier": {},
		"name":       {},
	},
}

type Project struct {
	Record
	Abstracts   []Attr[Text]       `json:"abstracts,omitempty"`
	Identifiers []Attr[Identifier] `json:"identifiers,omitempty"`
	Names       []Attr[Text]       `json:"names,omitempty"`
}

func loadProject(rec *DbRec) (*Project, error) {
	p := Project{}
	p.ID = rec.ID
	p.Kind = rec.Kind
	if err := loadAttrs(rec, "abstract", &p.Abstracts); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "identifier", &p.Identifiers); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "name", &p.Names); err != nil {
		return nil, err
	}
	return &p, nil
}
