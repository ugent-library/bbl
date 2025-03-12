package bbl

type RecordSpec struct {
	Kind     string               `json:"kind"`
	BaseKind string               `json:"-"`
	New      func() Record        `json:"-"`
	Attrs    map[string]*AttrSpec `json:"attrs"`
}

type AttrSpec struct {
	Use      bool          `json:"use"`
	Required bool          `json:"required"`
	Schemes  []*SchemeSpec `json:"schemes"`
}

type SchemeSpec struct {
	Scheme   string   `json:"scheme"`
	Required bool     `json:"required"`
	Codes    []string `json:"codes"`
}

type Record interface {
	Load(*RawRecord, map[string]*RecordSpec) error
	Validate() error
}

type RecordHeader struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind"`
}

func (r RecordHeader) IsNew() bool {
	return r.ID != ""
}
