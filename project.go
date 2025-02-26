package bbl

var projectKind = "project"

var projectSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"abstract":   {},
		"identifier": {},
		"name":       {},
	},
	Validate: func(dbr *DbRec) error {
		rec, err := loadProject(dbr)
		if err != nil {
			return err
		}
		return rec.Validate()
	},
}

type Project struct {
	Record
	Abstracts   []Attr[Text] `json:"abstracts,omitempty"`
	Identifiers []Attr[Code] `json:"identifiers,omitempty"`
	Names       []Attr[Text] `json:"names,omitempty"`
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

func (rec *Project) Validate() error {
	return nil
}
