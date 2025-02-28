package bbl

var personSpec = &RecordSpec{
	BaseKind: "person",
	New:      func() Record { return &Person{} },
	Attrs: map[string]*AttrSpec{
		"name_parts":           {},
		"preferred_name_parts": {},
	},
}

func loadPerson(rawRec *RawRecord) (*Person, error) {
	rec := &Person{}
	if err := rec.Load(rawRec); err != nil {
		return nil, err
	}
	return rec, nil
}

type Person struct {
	RecordHeader
	NameParts          *Attr[NameParts] `json:"name_parts,omitempty"`
	PreferredNameParts *Attr[NameParts] `json:"preferred_name_parts,omitempty"`
}

func (rec *Person) Load(rawRec *RawRecord) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
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
