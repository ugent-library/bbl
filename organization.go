package bbl

import (
	"slices"
)

type Organization struct {
	RecHeader
	Kind              string            `json:"kind"`
	Identifiers       []Code            `json:"identifiers,omitempty"`
	Rels              []OrganizationRel `json:"rels,omitempty"`
	OrganizationAttrs `json:"attrs"`
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
	// v := valgo.New()
	// v.Is(
	// 	valgo.String(rec.Kind, "kind").Not().Blank(),
	// 	valgo.Number(len(rec.Names), "names").Not().Zero(),
	// )
	// for i, ident := range rec.Identifiers {
	// 	v.InRow("identifiers", i, v.Is(
	// 		valgo.String(ident.Scheme, "scheme").Not().Blank(),
	// 		valgo.String(ident.Val, "val").Not().Blank(),
	// 	))
	// }
	// for i, rel := range rec.Rels {
	// 	v.InRow("rels", i, v.Is(
	// 		valgo.String(rel.Kind, "kind").Not().Blank(),
	// 		valgo.String(rel.OrganizationID, "organization_id").Not().Blank(),
	// 	))
	// }
	// return v.ToError()
}

func (rec *Organization) Diff(rec2 *Organization) map[string]any {
	changes := map[string]any{}
	if rec.Kind != rec2.Kind {
		changes["kind"] = rec.Kind
	}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if !slices.Equal(rec.Names, rec2.Names) {
		changes["names"] = rec.Names
	}
	if !slices.EqualFunc(rec.Rels, rec2.Rels, func(r1, r2 OrganizationRel) bool {
		return r1.Kind == r2.Kind && r1.OrganizationID == r2.OrganizationID
	}) {
		changes["rels"] = rec.Rels
	}
	return changes
}

func (rec *Organization) GetName() string {
	if len(rec.Names) > 0 {
		return rec.Names[0].Val
	}
	return ""
}
