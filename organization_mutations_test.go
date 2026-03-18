package bbl

import "testing"

func TestCreateOrganization_Apply(t *testing.T) {
	id := newID()
	m := &CreateOrganization{
		OrganizationID: id,
		Kind:     "department",
	}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	o := eff.record.(*Organization)
	if o.Version != 1 {
		t.Errorf("expected version 1, got %d", o.Version)
	}
	if o.Status != OrganizationStatusPublic {
		t.Errorf("expected status public, got %q", o.Status)
	}
	if o.Kind != "department" {
		t.Errorf("expected kind department, got %q", o.Kind)
	}
}

func TestDeleteOrganization_Apply(t *testing.T) {
	id := newID()
	existing := &Organization{
		ID:      id,
		Version: 1,
		Kind:    "department",
		Status:  OrganizationStatusPublic,
	}
	state := mutationState{organizations: map[ID]*Organization{id: existing}}

	m := &DeleteOrganization{OrganizationID: id}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	o := eff.record.(*Organization)
	if o.Status != OrganizationStatusDeleted {
		t.Errorf("expected deleted, got %q", o.Status)
	}
	if o.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
	if o.Version != 2 {
		t.Errorf("expected version 2, got %d", o.Version)
	}
}

func TestDeleteOrganization_AlreadyDeleted(t *testing.T) {
	id := newID()
	existing := &Organization{
		ID:      id,
		Version: 2,
		Kind:    "department",
		Status:  OrganizationStatusDeleted,
	}
	state := mutationState{organizations: map[ID]*Organization{id: existing}}

	m := &DeleteOrganization{OrganizationID: id}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted organization")
	}
}
