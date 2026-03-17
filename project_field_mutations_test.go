package bbl

import "testing"

func TestSetProjectTitles_Apply(t *testing.T) {
	projectID := newID()
	m := &SetProjectTitles{ProjectID: projectID, Titles: []Title{{Lang: "en", Val: "My Project"}}}
	eff, err := m.apply(mutationState{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if eff == nil {
		t.Fatal("expected non-nil effect")
	}
	if eff.recordType != RecordTypeProject {
		t.Errorf("expected RecordTypeProject, got %q", eff.recordType)
	}
	if eff.autoPin == nil {
		t.Error("expected autoPin to be set")
	}
}

func TestSetProjectDescriptions_Apply(t *testing.T) {
	projectID := newID()
	m := &SetProjectDescriptions{ProjectID: projectID, Descriptions: []Text{{Lang: "en", Val: "A description"}}}
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

func TestDeleteProjectDescriptions_Apply(t *testing.T) {
	projectID := newID()
	m := &DeleteProjectDescriptions{ProjectID: projectID}
	eff, err := m.apply(mutationState{}, nil)
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

func TestSetProjectIdentifiers_Apply(t *testing.T) {
	projectID := newID()
	m := &SetProjectIdentifiers{ProjectID: projectID, Identifiers: []Identifier{{Scheme: "iweto", Val: "P12345"}}}
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

func TestDeleteProjectIdentifiers_Apply(t *testing.T) {
	projectID := newID()
	m := &DeleteProjectIdentifiers{ProjectID: projectID}
	eff, err := m.apply(mutationState{}, nil)
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

func TestSetProjectPeople_Apply(t *testing.T) {
	projectID := newID()
	personID := newID()
	m := &SetProjectPeople{ProjectID: projectID, People: []ProjectPerson{{PersonID: personID, Role: "PI"}}}
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

func TestDeleteProjectPeople_Apply(t *testing.T) {
	projectID := newID()
	m := &DeleteProjectPeople{ProjectID: projectID}
	eff, err := m.apply(mutationState{}, nil)
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
