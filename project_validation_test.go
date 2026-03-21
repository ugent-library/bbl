package bbl

import "testing"

func TestProjectValidation_WithTitles(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("project", "")
	fields := map[string]any{"titles": []Title{{Val: "My Project"}}}
	errs := validateRecord("public", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestProjectValidation_MissingTitles(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("project", "")
	fields := map[string]any{}
	errs := validateRecord("public", fields, defs)
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error, got: %v", errs)
	}
}

func TestProjectValidation_EmptyTitles(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("project", "")
	fields := map[string]any{"titles": []Title{}}
	errs := validateRecord("public", fields, defs)
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error, got: %v", errs)
	}
}
