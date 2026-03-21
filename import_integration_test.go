package bbl

import (
	"context"
	"iter"
	"testing"
)

func TestImportWorksWithContributors(t *testing.T) {
	repo := testRepo(t)
	ctx := context.Background()

	if err := repo.UpsertSource(ctx, "test-source"); err != nil {
		t.Fatalf("upsert source: %v", err)
	}

	// Create a person to link as contributor.
	personID := createTestPerson(t, repo)

	records := []*ImportWorkInput{
		{
			SourceID:     "work-001",
			Kind:         "journal_article",
			Volume:       "42",
			SourceRecord: []byte(`{}`),
			Titles:   []Title{{Lang: "eng", Val: "Test Article"}},
			Contributors: []ImportWorkContributor{
				{
					PersonRef:  &Ref{ID: &personID},
					Kind:       "person",
					Name:       "Jane Doe",
					GivenName:  "Jane",
					FamilyName: "Doe",
					Roles:      []string{"author"},
				},
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

	n, err := repo.ImportWorks(ctx, "test-source", iter.Seq2[*ImportWorkInput, error](seq))
	if err != nil {
		t.Fatalf("import works: %v", err)
	}
	if n != 1 {
		t.Fatalf("imported %d works, want 1", n)
	}

	// Look up the work via the source table.
	var workID ID
	err = repo.db.QueryRow(ctx, `
		SELECT work_id FROM bbl_work_sources
		WHERE source = $1 AND source_id = $2`, "test-source", "work-001").Scan(&workID)
	if err != nil {
		t.Fatalf("lookup work by source: %v", err)
	}
	work, err := repo.GetWork(ctx, workID)
	if err != nil {
		t.Fatalf("get work: %v", err)
	}
	if work.Volume != "42" {
		t.Errorf("volume = %q, want %q", work.Volume, "42")
	}
	if len(work.Titles) != 1 {
		t.Fatalf("titles count = %d, want 1", len(work.Titles))
	}
	if work.Titles[0].Val != "Test Article" {
		t.Errorf("title = %q, want %q", work.Titles[0].Val, "Test Article")
	}
	if len(work.Contributors) != 1 {
		t.Fatalf("contributors count = %d, want 1", len(work.Contributors))
	}
	if work.Contributors[0].Name != "Jane Doe" {
		t.Errorf("contributor name = %q, want %q", work.Contributors[0].Name, "Jane Doe")
	}

	// Verify extension table has person_id.
	var extPersonID *ID
	err = repo.db.QueryRow(ctx, `
		SELECT c.person_id
		FROM bbl_work_assertion_contributors c
		JOIN bbl_work_assertions a ON a.id = c.assertion_id
		WHERE a.work_id = $1 AND a.pinned = true
		LIMIT 1`, work.ID).Scan(&extPersonID)
	if err != nil {
		t.Fatalf("query extension: %v", err)
	}
	if extPersonID == nil || *extPersonID != personID {
		t.Errorf("extension person_id = %v, want %s", extPersonID, personID)
	}
}
