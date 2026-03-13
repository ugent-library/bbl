package bbl

import "testing"

func TestSetWorkTitles_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkTitles{WorkID: workID, Titles: []Title{{Lang: "en", Val: "Test Title"}}}
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
	if eff.opType != OpUpdate {
		t.Errorf("expected OpUpdate, got %q", eff.opType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetWorkIdentifiers_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkIdentifiers{WorkID: workID, Identifiers: []WorkIdentifier{{Scheme: "doi", Val: "10.1234/test"}}}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestDeleteWorkIdentifiers_Apply(t *testing.T) {
	workID := newID()
	m := &DeleteWorkIdentifiers{WorkID: workID}
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
		t.Error("expected autoPin to be set")
	}
}

func TestSetWorkContributors_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkContributors{
		WorkID: workID,
		Contributors: []WorkContributor{
			{Name: "Jane Doe", GivenName: "Jane", FamilyName: "Doe", Roles: []string{"author"}},
		},
	}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestDeleteWorkContributors_Apply(t *testing.T) {
	workID := newID()
	m := &DeleteWorkContributors{WorkID: workID}
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
}

func TestSetWorkAbstracts_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkAbstracts{WorkID: workID, Abstracts: []Text{{Lang: "en", Val: "An abstract"}}}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestDeleteWorkAbstracts_Apply(t *testing.T) {
	workID := newID()
	m := &DeleteWorkAbstracts{WorkID: workID}
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
}

func TestSetWorkNotes_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkNotes{WorkID: workID, Notes: []Note{{Kind: "access", Val: "Open access"}}}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestSetWorkKeywords_Apply(t *testing.T) {
	workID := newID()
	m := &SetWorkKeywords{WorkID: workID, Keywords: []Keyword{{Val: "machine learning"}}}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestDeleteWorkKeywords_Apply(t *testing.T) {
	workID := newID()
	m := &DeleteWorkKeywords{WorkID: workID}
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
}

func TestSetWorkProjects_Apply(t *testing.T) {
	workID := newID()
	projectID := newID()
	m := &SetWorkProjects{WorkID: workID, Projects: []ID{projectID}}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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

func TestSetWorkRels_Apply(t *testing.T) {
	workID := newID()
	relatedID := newID()
	m := &SetWorkRels{
		WorkID: workID,
		Rels: []struct {
			RelatedWorkID ID
			Kind          string
		}{{RelatedWorkID: relatedID, Kind: "cites"}},
	}
	eff, err := m.apply(mutationState{}, AddRevInput{})
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
