package bbl

import (
	"slices"

	"github.com/ugent-library/vo"
)

type Organization struct {
	Header
	Kind              string            `json:"kind"`
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
	v := vo.New(
		vo.NotBlank("kind", rec.Kind),
		vo.NotEmpty("names", rec.Names),
	)

	for i, ident := range rec.Identifiers {
		v.In("identifiers").Index(i).Add(
			vo.NotBlank("scheme", ident.Scheme),
			vo.NotBlank("val", ident.Val),
		)
	}

	for i, text := range rec.Names {
		v.In("names").Index(i).Add(
			vo.ISO639_2("lang", text.Lang),
			vo.NotBlank("val", text.Val),
		)
	}

	return v.Validate().ToError()
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
