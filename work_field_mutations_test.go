package bbl

import "testing"

func TestSetWorkArticleNumber_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkArticleNumber{WorkID: workID, Val: "e12345"}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeWork {
		t.Errorf("expected RecordTypeWork, got %q", eff.recordType)
	}
	if eff.recordID != workID {
		t.Errorf("expected recordID %s, got %s", workID, eff.recordID)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetWorkConference_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkConference{
		WorkID: workID,
		Val:    Conference{Name: "ICSE 2024", Location: "Lisbon"},
	}
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

func TestSetWorkPages_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkPages{
		WorkID: workID,
		Val:    Extent{Start: "1", End: "42"},
	}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
}

func TestSetWorkVolume_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkVolume{WorkID: workID, Val: "42"}
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

func TestUnsetWorkVolume_Apply(t *testing.T) {
	workID := newID()
	m := &UnsetWorkVolume{WorkID: workID}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set for delete (re-evaluation)")
	}
}
