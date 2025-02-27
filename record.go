package bbl

import "strings"

type Record struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind"`
}

// TODO just use work profile
func (r Record) BaseKind() string {
	baseKind, _, _ := strings.Cut(r.Kind, ".")
	return baseKind
}

func (r Record) IsNew() bool {
	return r.ID != ""
}
