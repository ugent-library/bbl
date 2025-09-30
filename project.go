package bbl

import (
	"slices"
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
	return nil
	// v := valgo.New()
	// v.Is(
	// 	valgo.Number(len(rec.Names), "names").Not().Zero(),
	// )
	// for i, ident := range rec.Identifiers {
	// 	v.InRow("identifiers", i, v.Is(
	// 		valgo.String(ident.Scheme, "scheme").Not().Blank(),
	// 		valgo.String(ident.Val, "val").Not().Blank(),
	// 	))
	// }
	// return v.ToError()
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
