package bbl

import "testing"

func TestSetWorkArticleNumber_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkArticleNumber{WorkID: workID, Val: "e12345"}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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
	if eff.opType != OpUpdate {
		t.Errorf("expected OpUpdate, got %q", eff.opType)
	}
	if m.id == (ID{}) {
		t.Error("expected generated assertion ID")
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
	eff, err := m.apply(mutationState{}, AddRevInput{})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpUpdate {
		t.Errorf("expected OpUpdate, got %q", eff.opType)
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
	eff, err := m.apply(mutationState{}, AddRevInput{})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpUpdate {
		t.Errorf("expected OpUpdate, got %q", eff.opType)
	}
}

func TestSetWorkVolume_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkVolume{WorkID: workID, Val: "42"}
	eff, err := m.apply(mutationState{}, AddRevInput{})
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.opType != OpUpdate {
		t.Errorf("expected OpUpdate, got %q", eff.opType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestDeleteWorkVolume_Apply(t *testing.T) {
	workID := newID()
	m := &DeleteWorkVolume{WorkID: workID}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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
		t.Error("expected autoPin to be set for delete (re-evaluation)")
	}
}
