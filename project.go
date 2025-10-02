package bbl

import (
	"slices"

	"github.com/ugent-library/bbl/vo"
)

type Project struct {
	RecHeader
	Identifiers  []Code `json:"identifiers,omitempty"`
	ProjectAttrs `json:"attrs"`
}

type ProjectAttrs struct {
	Names     []Text `json:"names,omitempty"`
	Abstracts []Text `json:"abstracts,omitempty"`
}

func (rec *Project) Validate() error {
	v := vo.New(
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

func (rec *Project) Diff(rec2 *Project) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if !slices.Equal(rec.Names, rec2.Names) {
		changes["names"] = rec.Names
	}
	if !slices.Equal(rec.Abstracts, rec2.Abstracts) {
		changes["abstracts"] = rec.Abstracts
	}
	return changes
}

func (rec *Project) GetName() string {
	if len(rec.Names) > 0 {
		return rec.Names[0].Val
	}
	return ""
}
