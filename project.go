package bbl

import (
	"slices"
	"time"
)

type Project struct {
	ID        string       `json:"id,omitempty"`
	Source    string       `json:"source,omitempty"`
	SourceID  string       `json:"source_id,omitempty"`
	Attrs     ProjectAttrs `json:"attrs"`
	CreatedAt time.Time    `json:"created_at,omitzero"`
	UpdatedAt time.Time    `json:"updated_at,omitzero"`
}

type ProjectAttrs struct {
	Identifiers []Code `json:"identifiers,omitempty"`
	Names       []Text `json:"names,omitempty"`
	Abstracts   []Text `json:"abstracts,omitempty"`
}

func (rec *Project) RecID() string {
	return rec.ID
}

func (rec *Project) Diff(rec2 *Project) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Attrs.Identifiers, rec2.Attrs.Identifiers) {
		changes["identifiers"] = rec.Attrs.Identifiers
	}
	if !slices.Equal(rec.Attrs.Names, rec2.Attrs.Names) {
		changes["names"] = rec.Attrs.Names
	}
	if !slices.Equal(rec.Attrs.Abstracts, rec2.Attrs.Abstracts) {
		changes["abstracts"] = rec.Attrs.Abstracts
	}
	return changes
}
