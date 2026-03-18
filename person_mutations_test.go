package bbl

import "testing"

func TestCreatePerson_Apply(t *testing.T) {
	id := newID()
	m := &CreatePerson{PersonID: id}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	p := eff.record.(*Person)
	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
	if p.Status != PersonStatusPublic {
		t.Errorf("expected status public, got %q", p.Status)
	}
}

func TestDeletePerson_Apply(t *testing.T) {
	id := newID()
	existing := &Person{
		ID:      id,
		Version: 1,
		Status:  PersonStatusPublic,
	}
	state := mutationState{people: map[ID]*Person{id: existing}}

	m := &DeletePerson{PersonID: id}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	p := eff.record.(*Person)
	if p.Status != PersonStatusDeleted {
		t.Errorf("expected deleted, got %q", p.Status)
	}
	if p.DeletedAt == nil {
		t.Error("expected DeletedAt to be set")
	}
	if p.Version != 2 {
		t.Errorf("expected version 2, got %d", p.Version)
	}
}

func TestDeletePerson_AlreadyDeleted(t *testing.T) {
	id := newID()
	existing := &Person{
		ID:      id,
		Version: 2,
		Status:  PersonStatusDeleted,
	}
	state := mutationState{people: map[ID]*Person{id: existing}}

	m := &DeletePerson{PersonID: id}
	eff, err := m.apply(state, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff != nil {
		t.Fatal("expected nil (noop) for already-deleted person")
	}
}
