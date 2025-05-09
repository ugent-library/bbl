package bbl

import (
	"slices"
)

type Organization struct {
	RecHeader
	Kind        string            `json:"kind"`
	Identifiers []Code            `json:"identifiers,omitempty"`
	Rels        []OrganizationRel `json:"rels,omitempty"`
	Attrs       OrganizationAttrs `json:"attrs"`
}

type OrganizationAttrs struct {
	Names []Text `json:"names,omitempty"`
}

type OrganizationRel struct {
	Kind           string        `json:"kind"`
	OrganizationID string        `json:"organization_id"`
	Organization   *Organization `json:"organization,omitempty"`
}

func (rec *Organization) Validate() error {
	return nil
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
