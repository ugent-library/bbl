package bbl

import (
	"strings"
	"testing"

	"github.com/ugent-library/vo"
)

func testProfiles(t *testing.T) *Profiles {
	t.Helper()
	p, err := LoadProfiles("testdata/profiles.yaml")
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}
	return p
}

func TestWorkValidation_PrivateMissingAlwaysRequired(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("work", "journal_article")
	fields := map[string]any{}
	errs := validateRecord("private", fields, defs)
	// titles is required: always, so even private works need it
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error for private work, got: %v", errs)
	}
}

func TestWorkValidation_PrivateWithTitles(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("work", "journal_article")
	fields := map[string]any{
		"titles": []Title{{Lang: "eng", Val: "A title"}},
	}
	errs := validateRecord("private", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors for private work with titles, got: %v", errs)
	}
}

func TestWorkValidation_PublicMissingRequired(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("work", "journal_article")
	fields := map[string]any{}
	errs := validateRecord("public", fields, defs)
	if errs == nil {
		t.Fatal("expected errors for public work missing required fields")
	}
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error, got: %v", errs)
	}
	if !hasError(errs, "journal_title", "not_empty") {
		t.Errorf("expected journal_title not_empty error, got: %v", errs)
	}
}

func TestWorkValidation_PublicWithRequired(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("work", "journal_article")
	fields := map[string]any{
		"titles":           []Title{{Lang: "eng", Val: "A title"}},
		"journal_title":    "Test Journal",
		"publication_year": "2024",
	}
	errs := validateRecord("public", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

// hasError checks if any error matches the given path and rule.
func hasError(errs []*vo.Error, path, rule string) bool {
	for _, e := range errs {
		if e.Path == path && strings.Contains(e.Rule, rule) {
			return true
		}
	}
	return false
}
