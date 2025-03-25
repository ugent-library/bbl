package bbl

import (
	"slices"
	"time"
)

type Organization struct {
	ID        string            `json:"id,omitempty"`
	Kind      string            `json:"kind"`
	Attrs     OrganizationAttrs `json:"attrs"`
	Rels      []OrganizationRel `json:"rels,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitzero"`
	UpdatedAt time.Time         `json:"updated_at,omitzero"`
}

type OrganizationAttrs struct {
	Identifiers []Code `json:"identifiers,omitempty"`
	Names       []Text `json:"names,omitempty"`
}

type OrganizationRel struct {
	ID             string        `json:"id,omitempty"`
	Kind           string        `json:"kind"`
	OrganizationID string        `json:"organization_id"`
	Organization   *Organization `json:"organization,omitempty"`
}

func (rec *Organization) Diff(otherRec *Organization) map[string]any {
	changes := map[string]any{}
	if rec.Kind != otherRec.Kind {
		changes["kind"] = rec.Kind
	}
	if !slices.Equal(rec.Attrs.Identifiers, otherRec.Attrs.Identifiers) {
		changes["identifiers"] = rec.Attrs.Identifiers
	}
	if !slices.Equal(rec.Attrs.Names, otherRec.Attrs.Names) {
		changes["names"] = rec.Attrs.Names
	}
	if !slices.EqualFunc(rec.Rels, otherRec.Rels, func(rel, otherRel OrganizationRel) bool {
		return rel.Kind == otherRel.Kind && rel.OrganizationID == otherRel.OrganizationID
	}) {
		changes["rels"] = rec.Rels
	}
	return changes
}
