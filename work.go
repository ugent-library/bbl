package bbl

var workKind = "work"

var workSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"abstract":   {},
		"conference": {},
		"project":    {},
	},
}

type Work struct {
	ID         string                    `json:"id,omitempty"`
	Kind       string                    `json:"kind"`
	Abstracts  []Attr[Text]              `json:"abstracts,omitempty"`
	Conference *Attr[Conference]         `json:"conference,omitempty"`
	Projects   []RelAttr[Empty, Project] `json:"projects,omitempty"`
}

func loadWork(rec *DbRec) (*Work, error) {
	w := Work{}
	w.ID = rec.ID
	w.Kind = rec.Kind
	if err := loadAttrs(rec, "abstract", &w.Abstracts); err != nil {
		return nil, err
	}
	if err := loadAttr(rec, "conference", w.Conference); err != nil {
		return nil, err
	}
	if err := loadRelAttrs(rec, "project", &w.Projects); err != nil {
		return nil, err
	}
	return &w, nil
}
