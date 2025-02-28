package bbl

import "time"

var organizationSpec = &RecordSpec{
	BaseKind: "organization",
	New:      func() Record { return &Organization{} },
	Attrs: map[string]*AttrSpec{
		"ceased_on": {},
		"name":      {},
	},
}

func loadOrganization(rawRec *RawRecord) (*Organization, error) {
	rec := &Organization{}
	if err := rec.Load(rawRec); err != nil {
		return nil, err
	}
	return rec, nil
}

type Organization struct {
	RecordHeader
	CeasedOn *Attr[time.Time] `json:"ceased_on,omitempty"`
	Names    []Attr[Text]     `json:"names,omitempty"`
}

func (rec *Organization) Load(rawRec *RawRecord) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
	if err := loadAttr(rawRec, "ceased_on", &rec.CeasedOn); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "name", &rec.Names); err != nil {
		return err
	}
	return nil
}

func (rec *Organization) Validate() error {
	return nil
}
