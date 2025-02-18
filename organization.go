package bbl

import "time"

var organizationKind = "organization"

var organizationSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"ceased_on": {},
		"name":      {},
	},
}

type Organization struct {
	ID       string           `json:"id,omitempty"`
	Kind     string           `json:"kind"`
	CeasedOn *Attr[time.Time] `json:"ceased_on,omitempty"`
	Names    []Attr[Text]     `json:"names,omitempty"`
}

func loadOrganization(rec *DbRec) (*Organization, error) {
	o := Organization{}
	o.ID = rec.ID
	o.Kind = rec.Kind
	if err := loadAttr(rec, "ceased_on", &o.CeasedOn); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "name", &o.Names); err != nil {
		return nil, err
	}
	return &o, nil
}
