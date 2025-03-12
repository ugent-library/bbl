package bbl

import (
	"fmt"
	"time"
)

var organizationSpec = &RecordSpec{
	Kind:     "organization",
	BaseKind: "organization",
	New:      func() Record { return &Organization{} },
	Attrs: map[string]*AttrSpec{
		"ceased_on": {},
		"name":      {},
	},
}

func loadOrganization(rawRec *RawRecord, specMap map[string]*RecordSpec) (*Organization, error) {
	rec := &Organization{}
	if err := rec.Load(rawRec, specMap); err != nil {
		return nil, err
	}
	return rec, nil
}

type Organization struct {
	Spec *RecordSpec
	RecordHeader
	CeasedOn *Attr[time.Time] `json:"ceased_on,omitempty"`
	Names    []Attr[Text]     `json:"names,omitempty"`
}

func (rec *Organization) Load(rawRec *RawRecord, specMap map[string]*RecordSpec) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind
	spec, ok := specMap[rec.Kind]
	if !ok {
		return fmt.Errorf("spec not found: %s", rec.Kind)
	}
	rec.Spec = spec

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
