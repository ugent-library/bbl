package bbl

var workKind = "work"

var workSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"note":        {},
		"abstract":    {},
		"conference":  {},
		"contributor": {},
		"lay_summary": {},
		"project":     {},
		"title":       {},
	},
}

type Work struct {
	ID           string                          `json:"id,omitempty"`
	Kind         string                          `json:"kind"`
	Notes        []Attr[Note]                    `json:"notes,omitempty"`
	Abstracts    []Attr[Text]                    `json:"abstracts,omitempty"`
	Conference   *Attr[Conference]               `json:"conference,omitempty"`
	Contributors []RelAttr[Contributor, *Person] `json:"contributors,omitempty"`
	LaySummaries []Attr[Text]                    `json:"lay_summaries,omitempty"`
	Projects     []RelAttr[Empty, *Project]      `json:"projects,omitempty"`
	Titles       []Attr[Text]                    `json:"titles,omitempty"`
}

func loadWork(rec *DbRec) (*Work, error) {
	w := Work{}
	w.ID = rec.ID
	w.Kind = rec.Kind
	if err := loadAttrs(rec, "note", &w.Notes); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "abstract", &w.Abstracts); err != nil {
		return nil, err
	}
	if err := loadAttr(rec, "conference", &w.Conference); err != nil {
		return nil, err
	}
	if err := loadRelAttrs(rec, "contributor", &w.Contributors, loadPerson); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "lay_summary", &w.LaySummaries); err != nil {
		return nil, err
	}
	if err := loadRelAttrs(rec, "project", &w.Projects, loadProject); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "title", &w.Titles); err != nil {
		return nil, err
	}
	return &w, nil
}
