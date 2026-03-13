package bbl

import (
	"strings"
	"testing"

	"github.com/ugent-library/vo"
)

func testProfiles(t *testing.T) *WorkProfiles {
	t.Helper()
	p, err := LoadWorkProfiles("testdata/profiles.yaml")
	if err != nil {
		t.Fatalf("LoadWorkProfiles: %v", err)
	}
	return p
}

func TestValidateWork_UnknownKind(t *testing.T) {
	p := testProfiles(t)
	w := &Work{Kind: "unknown_kind"}
	errs := ValidateWork(w, p)
	if errs == nil {
		t.Fatal("expected errors")
	}
	if !hasError(errs, "kind", "one_of") {
		t.Errorf("expected kind one_of error, got: %v", errs)
	}
}

func TestValidateWork_ValidMinimal(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind:   "journal_article",
		Titles: []Title{{Lang: "eng", Val: "A title"}},
	}
	errs := ValidateWork(w, p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateWork_InvalidLang(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind:   "journal_article",
		Titles: []Title{{Lang: "xx", Val: "A title"}},
	}
	errs := ValidateWork(w, p)
	if !hasError(errs, "titles[0].lang", "iso639_2") {
		t.Errorf("expected lang error, got: %v", errs)
	}
}

func TestValidateWork_BlankTextVal(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind:   "journal_article",
		Titles: []Title{{Lang: "eng", Val: ""}},
	}
	errs := ValidateWork(w, p)
	if !hasError(errs, "titles[0].val", "not_blank") {
		t.Errorf("expected blank val error, got: %v", errs)
	}
}

func TestValidateWork_InvalidIdentifierScheme(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind: "journal_article",
		Identifiers: []WorkIdentifier{
			{Scheme: "isbn", Val: "978-0-123456-47-2"},
		},
	}
	errs := ValidateWork(w, p)
	if !hasError(errs, "identifiers[0].scheme", "one_of") {
		t.Errorf("expected scheme one_of error, got: %v", errs)
	}
}

func TestValidateWork_ValidIdentifierScheme(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind: "journal_article",
		Identifiers: []WorkIdentifier{
			{Scheme: "doi", Val: "10.1234/test"},
		},
	}
	errs := ValidateWork(w, p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateWork_MissingRequiredWhenPublic(t *testing.T) {
	t.Skip("TODO: rewrite for assertion model — scalar required field checks need str_fields")
}

func TestValidateWork_MissingRequiredWhenPrivate(t *testing.T) {
	p := testProfiles(t)
	w := &Work{
		Kind:   "journal_article",
		Status: WorkStatusPrivate,
	}
	errs := ValidateWork(w, p)
	if errs != nil {
		t.Errorf("expected no errors for private work, got: %v", errs)
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
