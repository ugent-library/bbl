package bbl

type Record struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind"`
}

func (r Record) IsNew() bool {
	return r.ID != ""
}
