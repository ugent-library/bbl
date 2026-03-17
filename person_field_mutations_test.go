package bbl

import "testing"

func TestSetPersonName_Apply(t *testing.T) {
	personID := newID()
	m := &SetPersonName{PersonID: personID, Val: "Jane Doe"}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypePerson {
		t.Errorf("expected RecordTypePerson, got %q", eff.recordType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetPersonGivenName_Apply(t *testing.T) {
	personID := newID()
	m := &SetPersonGivenName{PersonID: personID, Val: "Jane"}
	eff, err := m.apply(mutationState{}, nil)
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

func TestUnsetPersonGivenName_Apply(t *testing.T) {
	personID := newID()
	m := &UnsetPersonGivenName{PersonID: personID}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpDelete {
		t.Errorf("expected OpDelete, got %q", eff.opType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetPersonIdentifiers_Apply(t *testing.T) {
	personID := newID()
	m := &SetPersonIdentifiers{PersonID: personID, Identifiers: []Identifier{{Scheme: "orcid", Val: "0000-0001-2345-6789"}}}
	eff, err := m.apply(mutationState{}, nil)
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

func TestUnsetPersonIdentifiers_Apply(t *testing.T) {
	personID := newID()
	m := &UnsetPersonIdentifiers{PersonID: personID}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpDelete {
		t.Errorf("expected OpDelete, got %q", eff.opType)
	}
}

func TestSetPersonOrganizations_Apply(t *testing.T) {
	personID := newID()
	orgID := newID()
	m := &SetPersonOrganizations{PersonID: personID, Organizations: []PersonOrganization{{OrganizationID: orgID}}}
	eff, err := m.apply(mutationState{}, nil)
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

func TestUnsetPersonOrganizations_Apply(t *testing.T) {
	personID := newID()
	m := &UnsetPersonOrganizations{PersonID: personID}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpDelete {
		t.Errorf("expected OpDelete, got %q", eff.opType)
	}
}
