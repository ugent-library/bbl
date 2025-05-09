package bbl

import (
	"slices"
)

type Project struct {
	RecHeader
	Identifiers []Code       `json:"identifiers,omitempty"`
	Attrs       ProjectAttrs `json:"attrs"`
}

type ProjectAttrs struct {
	Names     []Text `json:"names,omitempty"`
	Abstracts []Text `json:"abstracts,omitempty"`
}

func (rec *Project) Validate() error {
	return nil
}

func (rec *Project) Diff(rec2 *Project) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if !slices.Equal(rec.Attrs.Names, rec2.Attrs.Names) {
		changes["names"] = rec.Attrs.Names
	}
	if !slices.Equal(rec.Attrs.Abstracts, rec2.Attrs.Abstracts) {
		changes["abstracts"] = rec.Attrs.Abstracts
	}
	return changes
}
