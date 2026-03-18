package bbl

import (
	"testing"
)

func TestCreateWork_Apply(t *testing.T) {
	id := newID()
	m := &CreateWork{
		WorkID: id,
		Kind:   "journal_article",
	}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeWork {
		t.Errorf("expected RecordTypeWork, got %q", eff.recordType)
	}
	w := eff.record.(*Work)
	if w.ID != id {
		t.Errorf("expected ID %s, got %s", id, w.ID)
	}
	if w.Version != 1 {
		t.Errorf("expected version 1, got %d", w.Version)
	}
	if w.Kind != "journal_article" {
		t.Errorf("expected kind journal_article, got %q", w.Kind)
	}
	if w.Status != WorkStatusPrivate {
		t.Errorf("expected status private, got %q", w.Status)
	}
}

func TestCreateWork_DefaultStatus(t *testing.T) {
	m := &CreateWork{
		WorkID: newID(),
		Kind:   "book",
	}
	eff, err := m.apply(updateState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := eff.record.(*Work)
	if w.Status != WorkStatusPrivate {
		t.Errorf("expected default status private, got %q", w.Status)
	}
}

func TestDeleteWork_Apply(t *testing.T) {
	id := newID()
	existing := &Work{
		ID:      id,
		Version: 1,
		Kind:    "journal_article",
		Status:  WorkStatusPublic,
	}
	state := updateState{works: map[ID]*Work{id: existing}}

	m := &DeleteWork{
		WorkID:     id,
		DeleteKind: WorkDeleteWithdrawn,
	}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	w := eff.record.(*Work)
	if w.Status != WorkStatusDeleted {
		t.Errorf("expected deleted status, got %q", w.Status)
	}
	if w.DeleteKind != WorkDeleteWithdrawn {
		t.Errorf("expected withdrawn, got %q", w.DeleteKind)
	}
	if w.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
	if w.Version != 2 {
		t.Errorf("expected version 2, got %d", w.Version)
	}
}

func TestDeleteWork_AlreadyDeleted(t *testing.T) {
	id := newID()
	existing := &Work{
		ID:      id,
		Version: 2,
		Kind:    "journal_article",
		Status:  WorkStatusDeleted,
	}
	state := updateState{works: map[ID]*Work{id: existing}}

	m := &DeleteWork{
		WorkID:     id,
		DeleteKind: WorkDeleteRetracted,
	}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted work")
	}
}
