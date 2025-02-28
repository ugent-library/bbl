package bbl

var projectSpec = &RecordSpec{
	BaseKind: "project",
	New:      func() Record { return &Project{} },
	Attrs: map[string]*AttrSpec{
		"abstract":   {},
		"identifier": {},
		"name":       {},
	},
}

func loadProject(rawRec *RawRecord) (*Project, error) {
	rec := &Project{}
	if err := rec.Load(rawRec); err != nil {
		return nil, err
	}
	return rec, nil
}

type Project struct {
	RecordHeader
	Abstracts   []Attr[Text] `json:"abstracts,omitempty"`
	Identifiers []Attr[Code] `json:"identifiers,omitempty"`
	Names       []Attr[Text] `json:"names,omitempty"`
}

func (rec *Project) Load(rawRec *RawRecord) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
	if err := loadAttrs(rawRec, "abstract", &rec.Abstracts); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "identifier", &rec.Identifiers); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "name", &rec.Names); err != nil {
		return err
	}
	return nil
}

func (rec *Project) Validate() error {
	return nil
}
