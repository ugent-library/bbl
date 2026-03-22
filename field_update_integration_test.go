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

// TestUpdateSetContributorsNoopWithSourceAssertions reproduces the scenario
// where source assertions with person_id exist alongside human assertions.
// When the source-pinned contributor has person_id from the extension table,
// re-submitting the same contributor (with person_id) must be detected as noop.
func TestUpdateSetContributorsNoopWithSourceAssertions(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	if err := repo.UpsertSource(ctx, "test-source"); err != nil {
		t.Fatalf("upsert source: %v", err)
	}

	personID := createTestPerson(t, repo)

	// Import a work with a contributor linked to a person.
	records := []*ImportWorkInput{
		{
			SourceID:     "work-001",
			Kind:         "journal_article",
			SourceRecord: []byte(`{}`),
			Titles:       []Title{{Lang: "eng", Val: "Test Article"}},
			Contributors: []ImportWorkContributor{
				{PersonRef: &Ref{ID: &personID}, Kind: "person", Name: "Jane Doe", GivenName: "Jane", FamilyName: "Doe", Roles: []string{"author"}},
			},
		},
	}
	seq := func(yield func(*ImportWorkInput, error) bool) {
		for _, r := range records {
			if !yield(r, nil) {
				return
			}
		}
	}
	n, err := repo.ImportWorks(ctx, "test-source", seq)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported %d, want 1", n)
	}

	var workID ID
	if err := repo.db.QueryRow(ctx, `
		SELECT work_id FROM bbl_work_sources
		WHERE source = 'test-source' AND source_id = 'work-001'`).Scan(&workID); err != nil {
		t.Fatalf("lookup: %v", err)
	}

	// Verify the cached contributor has person_id from enrichment.
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if len(work.Contributors) != 1 {
		t.Fatalf("contributors count = %d, want 1", len(work.Contributors))
	}
	if work.Contributors[0].PersonID == nil || *work.Contributors[0].PersonID != personID {
		t.Fatalf("cached person_id = %v, want %s", work.Contributors[0].PersonID, personID)
	}

	// Human sets the same contributor (including person_id from the form).
	contributors := []WorkContributor{
		{Kind: "person", Name: "Jane Doe", GivenName: "Jane", FamilyName: "Doe", PersonID: &personID, Roles: []string{"author"}},
	}

	// First human set — source already has same values, should be noop.
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if ok {
		t.Error("set contributors: source already has same values, expected noop")
	}
}

// TestUpdateSetContributorsNoopPersonIDMissing demonstrates the bug:
// source has person_id linked, form re-submits without person_id → should still be noop?
// Currently fails because the cached value has person_id but the submitted value doesn't.
// TestUpdateSetContributorsNoopAfterImportWithPersonRef tests the seed scenario:
// import with PersonRef (resolves to person_id + name), form re-submits same values.
func TestUpdateSetContributorsNoopAfterImportWithPersonRef(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	if err := repo.UpsertSource(ctx, "test-source"); err != nil {
		t.Fatalf("upsert source: %v", err)
	}

	// Create a person (like the seed does).
	personID := createTestPerson(t, repo)
	// Set name on the person so resolvePersonRef can fill contributor names.
	repo.db.Exec(ctx, `UPDATE bbl_people SET cache = $1 WHERE id = $2`,
		[]byte(`{"name":"Albert Einstein","given_name":"Albert","family_name":"Einstein"}`), personID)

	// Import a work with PersonRef (no explicit name — will be resolved from person).
	records := []*ImportWorkInput{
		{
			SourceID:     "w-001",
			Kind:         "journal_article",
			SourceRecord: []byte(`{}`),
			Titles:       []Title{{Lang: "eng", Val: "Test"}},
			Contributors: []ImportWorkContributor{
				{PersonRef: &Ref{ID: &personID}, Kind: "person", Roles: []string{"author"}},
			},
		},
	}
	seq := func(yield func(*ImportWorkInput, error) bool) {
		for _, r := range records {
			if !yield(r, nil) {
				return
			}
		}
	}
	n, err := repo.ImportWorks(ctx, "test-source", seq)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported %d, want 1", n)
	}

	var workID ID
	if err := repo.db.QueryRow(ctx, `
		SELECT work_id FROM bbl_work_sources
		WHERE source = 'test-source' AND source_id = 'w-001'`).Scan(&workID); err != nil {
		t.Fatalf("lookup: %v", err)
	}

	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	t.Logf("cached contributor: %+v", work.Contributors[0])

	// Simulate form re-submit: same values including person_id.
	contributors := work.Contributors // exact round-trip from cache
	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if ok {
		t.Error("set contributors: expected noop (same as source)")
	}
}

// TestUpdateSetContributorsNoopAfterImportNoPersonID tests the scenario where
// source imports contributors without person_id, and the form re-submits the same values.
func TestUpdateSetContributorsNoopAfterImportNoPersonID(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()
	user := createTestUser(t, repo, RoleUser)

	if err := repo.UpsertSource(ctx, "test-source"); err != nil {
		t.Fatalf("upsert source: %v", err)
	}

	// Import with contributors (no person link).
	records := []*ImportWorkInput{
		{
			SourceID:     "work-002",
			Kind:         "journal_article",
			SourceRecord: []byte(`{}`),
			Titles:       []Title{{Lang: "eng", Val: "Test"}},
			Contributors: []ImportWorkContributor{
				{Kind: "person", Name: "Albert Einstein", GivenName: "Albert", FamilyName: "Einstein", Roles: []string{"author"}},
				{Kind: "person", Name: "Marie Curie", GivenName: "Marie", FamilyName: "Curie", Roles: []string{"author"}},
			},
		},
	}
	seq := func(yield func(*ImportWorkInput, error) bool) {
		for _, r := range records {
			if !yield(r, nil) {
				return
			}
		}
	}
	n, err := repo.ImportWorks(ctx, "test-source", seq)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported %d, want 1", n)
	}

	var workID ID
	if err := repo.db.QueryRow(ctx, `
		SELECT work_id FROM bbl_work_sources
		WHERE source = 'test-source' AND source_id = 'work-002'`).Scan(&workID); err != nil {
		t.Fatalf("lookup: %v", err)
	}

	// Verify the cache has the expected contributors.
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if len(work.Contributors) != 2 {
		t.Fatalf("contributors = %d, want 2", len(work.Contributors))
	}
	t.Logf("cached contributors[0]: %+v", work.Contributors[0])
	t.Logf("cached contributors[1]: %+v", work.Contributors[1])

	// Form re-submits the same contributors (as if user hit save without changes).
	contributors := []WorkContributor{
		{Kind: "person", Name: "Albert Einstein", GivenName: "Albert", FamilyName: "Einstein", Roles: []string{"author"}},
		{Kind: "person", Name: "Marie Curie", GivenName: "Marie", FamilyName: "Curie", Roles: []string{"author"}},
	}

	ok, _, err := repo.Update(ctx, user, &Set{RecordType: "work", RecordID: workID, Field: "contributors", Val: contributors})
	if err != nil {
		t.Fatalf("set contributors: %v", err)
	}
	if ok {
		t.Error("set contributors: source has same values, expected noop")
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
