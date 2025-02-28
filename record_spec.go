package bbl

type RecordSpec struct {
	Kind     string               `json:"kind"`
	BaseKind string               `json:"-"`
	New      func() Record        `json:"-"`
	Attrs    map[string]*AttrSpec `json:"attrs"`
}

type AttrSpec struct {
	Use      bool `json:"use"`
	Required bool `json:"required"`
	// Vals     map[string]*ValSpec `json:"vals"`
}

// type ValSpec struct {
// }
