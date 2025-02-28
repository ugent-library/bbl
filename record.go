package bbl

import (
	"strings"
)

type Record interface {
	Load(*RawRecord) error
	Validate() error
}

type RecordHeader struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind"`
}

// TODO just use work profile
func (r RecordHeader) BaseKind() string {
	baseKind, _, _ := strings.Cut(r.Kind, ".")
	return baseKind
}

func (r RecordHeader) IsNew() bool {
	return r.ID != ""
}
