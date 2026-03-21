package bbl

import (
	"context"
	"testing"
)

func TestUpdateSetScalarField(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	// Create a work.
	workID := newID()
	ok, _, err := repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}
	if !ok {
		t.Fatal("create work: expected rev to be written")
	}

	// Set volume.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "volume", Val: "42"})
	if err != nil {
		t.Fatalf("set volume: %v", err)
	}
	if !ok {
		t.Fatal("set volume: expected rev to be written")
	}

	// Verify via GetWork.
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if work.Volume != "42" {
		t.Errorf("volume = %q, want %q", work.Volume, "42")
	}

	// Set again with same value — should be noop.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "volume", Val: "42"})
	if err != nil {
		t.Fatalf("set volume noop: %v", err)
	}
	if ok {
		t.Error("set volume noop: expected no rev (noop)")
	}
}

func TestUpdateSetTitles(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	titles := []Title{
		{Lang: "eng", Val: "A Great Paper"},
		{Lang: "fra", Val: "Un Grand Article"},
	}
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "titles", Val: titles})
	if err != nil {
		t.Fatalf("set titles: %v", err)
	}
	if !ok {
		t.Fatal("set titles: expected rev")
	}

	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if len(work.Titles) != 2 {
		t.Fatalf("titles count = %d, want 2", len(work.Titles))
	}
	if work.Titles[0].Val != "A Great Paper" {
		t.Errorf("title[0] = %q, want %q", work.Titles[0].Val, "A Great Paper")
	}
}

func TestUpdateSetContributors(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	personID := createTestPerson(t, repo)

	workID := newID()
	_, _, err := repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}

	// Set contributors with a linked person.
	contributors := []WorkContributor{
		{
			Kind:       "person",
			Name:       "Jane Doe",
			GivenName:  "Jane",
			FamilyName: "Doe",
			PersonID:   &personID,
			Roles:      []string{"author"},
		},
		{
			Kind:       "person",
			Name:       "Bob Smith",
			GivenName:  "Bob",
			FamilyName: "Smith",
			Roles:      []string{"editor"},
		},
	}
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if !ok {
		t.Fatal("set contributors: expected rev")
	}

	// Verify cache has contributors.
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if len(work.Contributors) != 2 {
		t.Fatalf("contributors count = %d, want 2", len(work.Contributors))
	}
	if work.Contributors[0].Name != "Jane Doe" {
		t.Errorf("contributor[0].Name = %q, want %q", work.Contributors[0].Name, "Jane Doe")
	}

	// Verify extension table has person_id by checking the assertion row.
	var extPersonID *ID
	err = repo.db.QueryRow(ctx, `
		SELECT c.person_id
		FROM bbl_work_assertion_contributors c
		JOIN bbl_work_assertions a ON a.id = c.assertion_id
		WHERE a.work_id = $1 AND a.pinned = true
		ORDER BY a.id
		LIMIT 1`, workID).Scan(&extPersonID)
	if err != nil {
		t.Fatalf("query extension: %v", err)
	}
	if extPersonID == nil || *extPersonID != personID {
		t.Errorf("extension person_id = %v, want %s", extPersonID, personID)
	}
}

func TestUpdateSetWorkProjects(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	projectID := createTestProject(t, repo)

	workID := newID()
	_, _, err := repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})
	if err != nil {
		t.Fatalf("create work: %v", err)
	}

	// Link work to project.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "projects", Val: []ID{projectID}})
	if err != nil {
		t.Fatalf("set projects: %v", err)
	}
	if !ok {
		t.Fatal("set projects: expected rev")
	}

	// Verify extension table.
	var extProjectID ID
	err = repo.db.QueryRow(ctx, `
		SELECT p.project_id
		FROM bbl_work_assertion_projects p
		JOIN bbl_work_assertions a ON a.id = p.assertion_id
		WHERE a.work_id = $1 AND a.pinned = true
		LIMIT 1`, workID).Scan(&extProjectID)
	if err != nil {
		t.Fatalf("query extension: %v", err)
	}
	if extProjectID != projectID {
		t.Errorf("extension project_id = %s, want %s", extProjectID, projectID)
	}
}

func TestUpdateUnsetField(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})
	repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "volume", Val: "10"})

	// Unset the field.
	ok, _, err := repo.Update(ctx, user, &Unset{RecordType: "work", RecordID: workID, Field: "volume"})
	if err != nil {
		t.Fatalf("unset volume: %v", err)
	}
	if !ok {
		t.Fatal("unset volume: expected rev")
	}

	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if work.Volume != "" {
		t.Errorf("volume = %q, want empty after unset", work.Volume)
	}
}

func TestUpdateSetContributorsNoop(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	contributors := []WorkContributor{
		{Kind: "person", Name: "Jane Doe", GivenName: "Jane", FamilyName: "Doe", Roles: []string{"author"}},
		{Kind: "person", Name: "Bob Smith", GivenName: "Bob", FamilyName: "Smith", Roles: []string{"editor"}},
	}

	// First set.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if !ok {
		t.Fatal("set contributors: expected rev")
	}

	// Same value again — should be noop.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors noop: %v", err)
	}
	if ok {
		t.Error("set contributors noop: expected no rev")
	}
}

func TestUpdateSetContributorsWithPersonIDNoop(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	personID := createTestPerson(t, repo)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	contributors := []WorkContributor{
		{Kind: "person", Name: "Jane Doe", GivenName: "Jane", FamilyName: "Doe", PersonID: &personID, Roles: []string{"author"}},
	}

	// First set.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if !ok {
		t.Fatal("set contributors: expected rev")
	}

	// Same value again — should be noop.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors noop: %v", err)
	}
	if ok {
		t.Error("set contributors with person_id noop: expected no rev")
	}
}

func TestUpdateSetWorkProjectsNoop(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	projectID := createTestProject(t, repo)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	// First set.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "projects", Val: []ID{projectID}})
	if err != nil {
		t.Fatalf("set projects: %v", err)
	}
	if !ok {
		t.Fatal("set projects: expected rev")
	}

	// Same value again — should be noop.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "projects", Val: []ID{projectID}})
	if err != nil {
		t.Fatalf("set projects noop: %v", err)
	}
	if ok {
		t.Error("set projects noop: expected no rev")
	}
}

func TestUpdateSetTitlesNoop(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	titles := []Title{{Lang: "eng", Val: "A Great Paper"}}

	// First set.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "titles", Val: titles})
	if err != nil {
		t.Fatalf("set titles: %v", err)
	}
	if !ok {
		t.Fatal("set titles: expected rev")
	}

	// Same value again — should be noop.
	ok, _, err = repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "titles", Val: titles})
	if err != nil {
		t.Fatalf("set titles noop: %v", err)
	}
	if ok {
		t.Error("set titles noop: expected no rev")
	}
}

func TestUpdateHideField(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	workID := newID()
	repo.Update(ctx, user, &CreateWork{ID: workID, Kind: "journal_article"})

	ok, _, err := repo.Update(ctx, user, &Hide{RecordType: "work", RecordID: workID, Field: "volume"})
	if err != nil {
		t.Fatalf("hide volume: %v", err)
	}
	if !ok {
		t.Fatal("hide volume: expected rev")
	}

	// Hide again — should be noop.
	ok, _, err = repo.Update(ctx, user, &Hide{RecordType: "work", RecordID: workID, Field: "volume"})
	if err != nil {
		t.Fatalf("hide volume noop: %v", err)
	}
	if ok {
		t.Error("hide volume noop: expected no rev")
	}
}
