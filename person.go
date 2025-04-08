package bbl

import (
	"slices"
	"time"
)

type Person struct {
	ID        string      `json:"id,omitempty"`
	Source    string      `json:"source,omitempty"`
	SourceID  string      `json:"source_id,omitempty"`
	Attrs     PersonAttrs `json:"attrs"`
	CreatedAt time.Time   `json:"created_at,omitzero"`
	UpdatedAt time.Time   `json:"updated_at,omitzero"`
}

type PersonAttrs struct {
	Identifiers []Code `json:"identifiers,omitempty"`
	Name        string `json:"name"`
	GivenName   string `json:"given_name,omitempty"`
	MiddleName  string `json:"middle_name,omitempty"`
	FamilyName  string `json:"family_name,omitempty"`
}

func (rec *Person) RecID() string {
	return rec.ID
}

func (rec *Person) Diff(rec2 *Person) map[string]any {
	changes := map[string]any{}
	if !slices.Equal(rec.Attrs.Identifiers, rec2.Attrs.Identifiers) {
		changes["identifiers"] = rec.Attrs.Identifiers
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
