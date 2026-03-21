package bbl

import (
	"context"
	"os"
	"testing"
)

// testDSN returns the database connection string for integration tests.
// Set BBL_TEST_DSN to override. Skips the test if no DSN is available.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("BBL_TEST_DSN")
	if dsn == "" {
		dsn = "postgres://bbl:bbl@localhost:3351/bbl_test"
	}
	return dsn
}

// testRepo creates a Repo for integration tests, runs migrations,
// and truncates all tables before returning. Cleans up on test end.
func testRepo(t *testing.T) *Repo {
	t.Helper()
	ctx := context.Background()

	dsn := testDSN(t)

	if err := MigrateUp(ctx, dsn); err != nil {
		t.Skipf("skipping integration test: migrate up: %v", err)
	}

	repo, err := NewRepo(ctx, dsn, make([]byte, 32))
	if err != nil {
		t.Skipf("skipping integration test: connect: %v", err)
	}
	t.Cleanup(func() { repo.Close() })

	truncateTables(t, repo)
	return repo
}

func truncateTables(t *testing.T, repo *Repo) {
	t.Helper()
	ctx := context.Background()
	_, err := repo.db.Exec(ctx, `
		TRUNCATE
			bbl_revs,
			bbl_work_assertions,
			bbl_work_assertion_contributors,
			bbl_work_assertion_projects,
			bbl_work_assertion_organizations,
			bbl_work_assertion_rels,
			bbl_person_assertions,
			bbl_person_assertion_affiliations,
			bbl_project_assertions,
			bbl_project_assertion_participants,
			bbl_organization_assertions,
			bbl_organization_assertion_rels,
			bbl_works,
			bbl_people,
			bbl_projects,
			bbl_organizations,
			bbl_users,
			bbl_history
		CASCADE
	`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

// createTestUser creates a user with the given role for testing.
func createTestUser(t *testing.T, repo *Repo, role string) *User {
	t.Helper()
	u, err := repo.CreateUser(context.Background(), UserAttrs{
		Username: "test-" + role,
		Email:    role + "@test.local",
		Name:     "Test " + role,
		Role:     role,
	})
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}

// createTestPerson inserts a person row directly (bypassing validation).
func createTestPerson(t *testing.T, repo *Repo) ID {
	t.Helper()
	id := newID()
	_, err := repo.db.Exec(context.Background(), `
		INSERT INTO bbl_people (id, version, status) VALUES ($1, 1, 'public')`, id)
	if err != nil {
		t.Fatalf("create test person: %v", err)
	}
	return id
}

// createTestProject inserts a project row directly (bypassing validation).
func createTestProject(t *testing.T, repo *Repo) ID {
	t.Helper()
	id := newID()
	_, err := repo.db.Exec(context.Background(), `
		INSERT INTO bbl_projects (id, version, status) VALUES ($1, 1, 'public')`, id)
	if err != nil {
		t.Fatalf("create test project: %v", err)
	}
	return id
}
