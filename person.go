package bbl

import (
	"slices"
)

type Person struct {
	RecHeader
	Identifiers []Code      `json:"identifiers,omitempty"`
	Attrs       PersonAttrs `json:"attrs"`
}

type PersonAttrs struct {
	Name       string `json:"name"`
	GivenName  string `json:"given_name,omitempty"`
	MiddleName string `json:"middle_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
}

func (rec *Person) Validate() error {
	return nil
}

func (rec *Person) Diff(rec2 *Person) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Identifiers, rec2.Identifiers) {
		changes["identifiers"] = rec.Identifiers
	}
	if rec.Attrs.Name != rec2.Attrs.Name {
		changes["name"] = rec.Attrs.Name
	}
	if rec.Attrs.GivenName != rec2.Attrs.GivenName {
		changes["given_name"] = rec.Attrs.GivenName
	}
	if rec.Attrs.MiddleName != rec2.Attrs.MiddleName {
		changes["middle_name"] = rec.Attrs.MiddleName
	}
	if rec.Attrs.FamilyName != rec2.Attrs.FamilyName {
		changes["family_name"] = rec.Attrs.FamilyName
	}
	return changes
}
