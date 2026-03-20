package bbl

import (
	"testing"
	"time"
)

func TestCreateProject_Apply(t *testing.T) {
	id := newID()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	m := &CreateProject{
		ProjectID: id,
		StartDate: &start,
	}
	eff, err := m.apply(updateState{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	p := eff.record.(*Project)
	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
	if p.Status != ProjectStatusPublic {
		t.Errorf("expected default status public, got %q", p.Status)
	}
}

func TestDeleteProject_Apply(t *testing.T) {
	id := newID()
	existing := &Project{
		ID:      id,
		Version: 1,
		Status:  ProjectStatusPublic,
	}
	state := updateState{projects: map[ID]*Project{id: existing}}

	m := &DeleteProject{ProjectID: id}
	eff, err := m.apply(state, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	p := eff.record.(*Project)
	if p.Status != ProjectStatusDeleted {
		t.Errorf("expected deleted, got %q", p.Status)
	}
	if p.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
	if p.Version != 2 {
		t.Errorf("expected version 2, got %d", p.Version)
	}
}

func TestDeleteProject_AlreadyDeleted(t *testing.T) {
	id := newID()
	existing := &Project{
		ID:      id,
		Version: 2,
		Status:  ProjectStatusDeleted,
	}
	state := updateState{projects: map[ID]*Project{id: existing}}

	m := &DeleteProject{ProjectID: id}
	eff, err := m.apply(state, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted project")
	}
}
