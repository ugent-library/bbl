package bbl

import (
	"testing"
)

func TestCreateWork_Apply(t *testing.T) {
	id := newID()
	m := &CreateWork{
		ID:   id,
		Kind: "journal_article",
	}
	state := updateState{records: make(map[ID]*recordState)}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeWork {
		t.Errorf("expected RecordTypeWork, got %q", eff.recordType)
	}
	rs := state.records[id]
	if rs == nil {
		t.Fatal("expected recordState to be created")
	}
	if rs.kind != "journal_article" {
		t.Errorf("expected kind journal_article, got %q", rs.kind)
	}
	if rs.status != WorkStatusPrivate {
		t.Errorf("expected status private, got %q", rs.status)
	}
}

func TestCreateWork_DefaultStatus(t *testing.T) {
	m := &CreateWork{
		ID:   newID(),
		Kind: "book",
	}
	state := updateState{records: make(map[ID]*recordState)}
	_, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	rs := state.records[m.ID]
	if rs.status != WorkStatusPrivate {
		t.Errorf("expected default status private, got %q", rs.status)
	}
}

func TestDeleteWork_Apply(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {
			recordType: RecordTypeWork,
			id:         id,
			version:    1,
			status:     WorkStatusPublic,
			kind:       "journal_article",
			fields:     make(map[string]any),
			assertions: make(map[string][]assertion),
		},
	}}

	m := &DeleteWork{
		WorkID:     id,
		DeleteKind: WorkDeleteWithdrawn,
	}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeWork {
		t.Errorf("expected RecordTypeWork, got %q", eff.recordType)
	}
}

func TestDeleteWork_AlreadyDeleted(t *testing.T) {
	id := newID()
	state := updateState{records: map[ID]*recordState{
		id: {
			recordType: RecordTypeWork,
			id:         id,
			version:    2,
			status:     WorkStatusDeleted,
			kind:       "journal_article",
			fields:     make(map[string]any),
			assertions: make(map[string][]assertion),
		},
	}}

	m := &DeleteWork{
		WorkID:     id,
		DeleteKind: WorkDeleteRetracted,
	}
	eff, err := m.apply(state, &User{Role: RoleUser})
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted work")
	}
}
