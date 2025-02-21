package bbl

var personKind = "person"

var personSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"name_parts":           {},
		"preferred_name_parts": {},
	},
}

type Person struct {
	Record
	NameParts          *Attr[NameParts] `json:"name_parts,omitempty"`
	PreferredNameParts *Attr[NameParts] `json:"preferred_name_parts,omitempty"`
}

func loadPerson(rec *DbRec) (*Person, error) {
	p := Person{}
	p.ID = rec.ID
	p.Kind = rec.Kind
	if err := loadAttr(rec, "name_parts", &p.NameParts); err != nil {
		return nil, err
	}
	if err := loadAttr(rec, "preferred_name_parts", &p.PreferredNameParts); err != nil {
		return nil, err
	}
	return &p, nil
}
