package bbl

var workSpec = &RecordSpec{
	Kind:     "work",
	BaseKind: "work",
	New:      func() Record { return &Work{} },
	Attrs: map[string]*AttrSpec{
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
}

func loadWork(rawRec *RawRecord) (*Work, error) {
	rec := &Work{}
	if err := rec.Load(rawRec); err != nil {
		return nil, err
	}
	return rec, nil
}

type Work struct {
	Profile *WorkProfile `json:"-"`

	RecordHeader
	RecordIdentifiers
	Notes           []Attr[Note]                    `json:"notes,omitempty"`
	Abstracts       []Attr[Text]                    `json:"abstracts,omitempty"`
	Classifications []Attr[Code]                    `json:"classifications,omitempty"`
	Conference      *Attr[Conference]               `json:"conference,omitempty"`
	Contributors    []RelAttr[Contributor, *Person] `json:"contributors,omitempty"`
	Keywords        []Attr[Code]                    `json:"keywords,omitempty"`
	LaySummaries    []Attr[Text]                    `json:"lay_summaries,omitempty"`
	Projects        []RelAttr[Empty, *Project]      `json:"projects,omitempty"`
	Titles          []Attr[Text]                    `json:"titles,omitempty"`
}

func (rec *Work) Load(rawRec *RawRecord) error {
	rec.ID = rawRec.ID
	rec.Kind = rawRec.Kind

	rec.Profile = getWorkProfile(rec.Kind)

	if err := loadAttrs(rawRec, "note", &rec.Notes); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "abstract", &rec.Abstracts); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "classification", &rec.Classifications); err != nil {
		return err
	}
	if err := loadAttr(rawRec, "conference", &rec.Conference); err != nil {
		return err
	}
	if err := loadRelAttrs(rawRec, "contributor", &rec.Contributors, loadPerson); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "identifier", &rec.Identifiers); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "keyword", &rec.Keywords); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "lay_summary", &rec.LaySummaries); err != nil {
		return err
	}
	if err := loadRelAttrs(rawRec, "project", &rec.Projects, loadProject); err != nil {
		return err
	}
	if err := loadAttrs(rawRec, "title", &rec.Titles); err != nil {
		return err
	}

	return nil
}

func (rec *Work) Validate() error {
	return nil
}
