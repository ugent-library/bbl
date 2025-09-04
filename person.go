package bbl

import (
	"slices"
)

type Person struct {
	RecHeader
	Identifiers []Code `json:"identifiers,omitempty"`
	PersonAttrs `json:"attrs"`
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
	if rec.Name != rec2.Name {
		changes["name"] = rec.Name
	}
	if rec.GivenName != rec2.GivenName {
		changes["given_name"] = rec.GivenName
	}
	if rec.MiddleName != rec2.MiddleName {
		changes["middle_name"] = rec.MiddleName
	}
	if rec.FamilyName != rec2.FamilyName {
		changes["family_name"] = rec.FamilyName
	}
	return changes
}
