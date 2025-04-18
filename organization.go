package bbl

import (
	"slices"
	"time"
)

type Organization struct {
	ID          string            `json:"id,omitempty"`
	Kind        string            `json:"kind"`
	Identifiers []Code            `json:"identifiers,omitempty"`
	Rels        []OrganizationRel `json:"rels,omitempty"`
	Attrs       OrganizationAttrs `json:"attrs"`
	CreatedAt   time.Time         `json:"created_at,omitzero"`
	UpdatedAt   time.Time         `json:"updated_at,omitzero"`
}

type OrganizationAttrs struct {
	Names []Text `json:"names,omitempty"`
}

type OrganizationRel struct {
	Kind           string        `json:"kind"`
	OrganizationID string        `json:"organization_id"`
	Organization   *Organization `json:"organization,omitempty"`
}

func (rec *Organization) RecID() string {
	return rec.ID
}

func (rec *Organization) Diff(rec2 *Organization) map[string]any {
	changes := map[string]any{}
	if rec.Kind != rec2.Kind {
		changes["kind"] = rec.Kind
	}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if !slices.Equal(rec.Attrs.Names, rec2.Attrs.Names) {
		changes["names"] = rec.Attrs.Names
	}
	if !slices.EqualFunc(rec.Rels, rec2.Rels, func(r1, r2 OrganizationRel) bool {
		return r1.Kind == r2.Kind && r1.OrganizationID == r2.OrganizationID
	}) {
		changes["rels"] = rec.Rels
	}
	return changes
}
