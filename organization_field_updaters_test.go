package bbl

import "testing"

func TestSetOrganizationNames_Apply(t *testing.T) {
	orgID := newID()
	m := &SetOrganizationNames{OrganizationID: orgID, Names: []Text{{Lang: "en", Val: "Ghent University"}}}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeOrganization {
		t.Errorf("expected RecordTypeOrganization, got %q", eff.recordType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetOrganizationIdentifiers_Apply(t *testing.T) {
	orgID := newID()
	m := &SetOrganizationIdentifiers{OrganizationID: orgID, Identifiers: []Identifier{{Scheme: "ror", Val: "https://ror.org/123"}}}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestUnsetOrganizationIdentifiers_Apply(t *testing.T) {
	orgID := newID()
	m := &UnsetOrganizationIdentifiers{OrganizationID: orgID}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}

func TestSetOrganizationRels_Apply(t *testing.T) {
	orgID := newID()
	relOrgID := newID()
	m := &SetOrganizationRels{
		OrganizationID: orgID,
		Rels: []struct {
			RelOrganizationID ID     `json:"rel_organization_id"`
			Kind              string `json:"kind"`
		}{{RelOrganizationID: relOrgID, Kind: "parent"}},
	}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestUnsetOrganizationRels_Apply(t *testing.T) {
	orgID := newID()
	m := &UnsetOrganizationRels{OrganizationID: orgID}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}
