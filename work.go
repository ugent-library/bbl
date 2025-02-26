package bbl

var workKind = "work"

var workSpec = &recSpec{
	Attrs: map[string]*attrSpec{
		"note":           {},
		"abstract":       {},
		"classification": {},
		"conference":     {},
		"contributor":    {},
		"identifier":     {},
		"keyword":        {},
		"lay_summary":    {},
		"project":        {},
		"title":          {},
	},
	Validate: func(dbr *DbRec) error {
		rec, err := loadWork(dbr)
		if err != nil {
			return err
		}
		return rec.Validate()
	},
}

type Work struct {
	Record
	Notes           []Attr[Note]                    `json:"notes,omitempty"`
	Abstracts       []Attr[Text]                    `json:"abstracts,omitempty"`
	Classifications []Attr[Code]                    `json:"classifications,omitempty"`
	Conference      *Attr[Conference]               `json:"conference,omitempty"`
	Contributors    []RelAttr[Contributor, *Person] `json:"contributors,omitempty"`
	Identifiers     []Attr[Code]                    `json:"identifiers,omitempty"`
	Keywords        []Attr[Code]                    `json:"keywords,omitempty"`
	LaySummaries    []Attr[Text]                    `json:"lay_summaries,omitempty"`
	Projects        []RelAttr[Empty, *Project]      `json:"projects,omitempty"`
	Titles          []Attr[Text]                    `json:"titles,omitempty"`
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
	if err := loadAttrs(rec, "classification", &w.Classifications); err != nil {
		return nil, err
	}
	if err := loadAttr(rec, "conference", &w.Conference); err != nil {
		return nil, err
	}
	if err := loadRelAttrs(rec, "contributor", &w.Contributors, loadPerson); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "identifier", &w.Identifiers); err != nil {
		return nil, err
	}
	if err := loadAttrs(rec, "keyword", &w.Keywords); err != nil {
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

func (rec *Work) Validate() error {
	return nil
}
