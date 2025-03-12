package bbl

import "fmt"

var personSpec = &RecordSpec{
	Kind:     "person",
	BaseKind: "person",
	New:      func() Record { return &Person{} },
	Attrs: map[string]*AttrSpec{
		"name_parts":           {},
		"preferred_name_parts": {},
	},
}

func loadPerson(rawRec *RawRecord, specMap map[string]*RecordSpec) (*Person, error) {
	rec := &Person{}
	if err := rec.Load(rawRec, specMap); err != nil {
		return nil, err
	}
	return rec, nil
}

type Person struct {
	Spec *RecordSpec
	RecordHeader
	NameParts          *Attr[NameParts] `json:"name_parts,omitempty"`
	PreferredNameParts *Attr[NameParts] `json:"preferred_name_parts,omitempty"`
}

func (rec *Person) Load(rawRec *RawRecord, specMap map[string]*RecordSpec) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
	spec, ok := specMap[rec.Kind]
	if !ok {
		return fmt.Errorf("spec not found: %s", rec.Kind)
	}
	rec.Spec = spec

	if err := loadAttr(rawRec, "name_parts", &rec.NameParts); err != nil {
		return err
	}
	if err := loadAttr(rawRec, "preferred_name_parts", &rec.PreferredNameParts); err != nil {
		return err
	}
	return nil
}

func (rec *Person) Validate() error {
	return nil
}
