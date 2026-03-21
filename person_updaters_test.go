package bbl

import "testing"

func TestCreatePerson_Apply(t *testing.T) {
	id := newID()
	m := &CreatePerson{ID: id}
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
	if rs.status != PersonStatusPublic {
		t.Errorf("expected status public, got %q", rs.status)
	}
}

func TestDeletePerson_Apply(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypePerson, id: id, version: 1, status: PersonStatusPublic,
			fields: make(map[string]any), assertions: make(map[string]*fieldState)},
	}}

	m := &DeletePerson{PersonID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}

func TestDeletePerson_AlreadyDeleted(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypePerson, id: id, version: 2, status: PersonStatusDeleted,
			fields: make(map[string]any), assertions: make(map[string]*fieldState)},
	}}

	m := &DeletePerson{PersonID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted person")
	}
}
