package bbl

import "testing"

func TestCreateOrganization_Apply(t *testing.T) {
	id := newID()
	m := &CreateOrganization{
		ID:   id,
		Kind: "department",
	}
	state := updateState{records: make(map[ID]*recordState)}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	rs := state.records[id]
	if rs == nil {
		t.Fatal("expected recordState")
	}
	if rs.status != OrganizationStatusPublic {
		t.Errorf("expected status public, got %q", rs.status)
	}
	if rs.kind != "department" {
		t.Errorf("expected kind department, got %q", rs.kind)
	}
}

func TestDeleteOrganization_Apply(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypeOrganization, id: id, version: 1, kind: "department", status: OrganizationStatusPublic,
			fields: make(map[string]any), assertions: make(map[string][]assertion)},
	}}

	m := &DeleteOrganization{OrganizationID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}

func TestDeleteOrganization_AlreadyDeleted(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypeOrganization, id: id, version: 2, kind: "department", status: OrganizationStatusDeleted,
			fields: make(map[string]any), assertions: make(map[string][]assertion)},
	}}

	m := &DeleteOrganization{OrganizationID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted organization")
	}
}
