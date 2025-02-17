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
	ID          string             `json:"id,omitempty"`
	Kind        string             `json:"kind"`
	Abstracts   []Attr[Text]       `json:"abstracts,omitempty"`
	Identifiers []Attr[Identifier] `json:"identifiers,omitempty"`
	Names       []Attr[Text]       `json:"names,omitempty"`
}

func loadProject(rec *DbRec) (*Project, error) {
	o := Project{}
	o.ID = rec.ID
	o.Kind = rec.Kind
	if err := loadAttrs(rec, "abstract", &o.Abstracts); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "identifier", &o.Identifiers); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "name", &o.Names); err != nil {
		return nil, err
	}
	return &o, nil
}
