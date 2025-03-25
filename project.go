package bbl

import (
	"slices"
	"time"
)

type Project struct {
	ID        string       `json:"id,omitempty"`
	Attrs     ProjectAttrs `json:"attrs"`
	CreatedAt time.Time    `json:"created_at,omitzero"`
	UpdatedAt time.Time    `json:"updated_at,omitzero"`
}

type ProjectAttrs struct {
	Identifiers []Code `json:"identifiers,omitempty"`
	Names       []Text `json:"names,omitempty"`
	Abstracts   []Text `json:"abstracts,omitempty"`
}

func (rec *Project) Diff(otherRec *Project) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Attrs.Identifiers, otherRec.Attrs.Identifiers) {
		changes["identifiers"] = rec.Attrs.Identifiers
	}
	if !slices.Equal(rec.Attrs.Names, otherRec.Attrs.Names) {
		changes["names"] = rec.Attrs.Names
	}
	if !slices.Equal(rec.Attrs.Abstracts, otherRec.Attrs.Abstracts) {
		changes["abstracts"] = rec.Attrs.Abstracts
	}
	return changes
}
