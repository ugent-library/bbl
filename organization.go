package bbl

// var OrganizationSpec = &RecSpec[*Organization]{
// 	Attrs: map[string]AttrSpec[*Organization]{
// 		"ceased_on": {
// 			Decode: decodeVal[time.Time],
// 			Reify: func(rec *Organization) {
// 				setAttr(rec, "ceased_on", rec.Attrs.CeasedOn)
// 			},
// 		},
// 		"name": {
// 			Decode: decodeVal[Text],
// 			Reify: func(rec *Organization) {
// 				setAttrs(rec, "name", &rec.Attrs.Names)
// 			},
// 		},
// 	},
// }

// type Organization = Rec[OrganizationAttrs]

// type OrganizationAttrs struct {
// 	CeasedOn *Attr[time.Time] `json:"ceased_on,omitempty"`
// 	Names    []Attr[Text]     `json:"names,omitempty"`
// }

// func NewOrganization(id, kind string) *Organization {
// 	rec := &Organization{}
// 	rec.change(AddRec(id, kind))
// 	return rec
// }
