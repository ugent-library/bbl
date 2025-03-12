package bbl

import "fmt"

var projectSpec = &RecordSpec{
	Kind:     "project",
	BaseKind: "project",
	New:      func() Record { return &Project{} },
	Attrs: map[string]*AttrSpec{
		"abstract":   {},
		"identifier": {},
		"name":       {},
	},
}

func loadProject(rawRec *RawRecord, specMap map[string]*RecordSpec) (*Project, error) {
	rec := &Project{}
	if err := rec.Load(rawRec, specMap); err != nil {
		return nil, err
	}
	return rec, nil
}

type Project struct {
	Spec *RecordSpec
	RecordHeader
	Abstracts   []Attr[Text] `json:"abstracts,omitempty"`
	Identifiers []Attr[Code] `json:"identifiers,omitempty"`
	Names       []Attr[Text] `json:"names,omitempty"`
}

func (rec *Project) Load(rawRec *RawRecord, specMap map[string]*RecordSpec) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
	spec, ok := specMap[rec.Kind]
	if !ok {
		return fmt.Errorf("spec not found: %s", rec.Kind)
	}
	rec.Spec = spec

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
