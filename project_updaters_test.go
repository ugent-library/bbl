package bbl

import "testing"

func TestCreateProject_Apply(t *testing.T) {
	id := newID()
	m := &CreateProject{ID: id}
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
	if rs.status != ProjectStatusPublic {
		t.Errorf("expected status public, got %q", rs.status)
	}
}

func TestDeleteProject_Apply(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypeProject, id: id, version: 1, status: ProjectStatusPublic,
			fields: make(map[string]any), assertions: make(map[string]*fieldState)},
	}}

	m := &DeleteProject{ProjectID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}

func TestDeleteProject_AlreadyDeleted(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {recordType: RecordTypeProject, id: id, version: 2, status: ProjectStatusDeleted,
			fields: make(map[string]any), assertions: make(map[string]*fieldState)},
	}}

	m := &DeleteProject{ProjectID: id}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted project")
	}
}
