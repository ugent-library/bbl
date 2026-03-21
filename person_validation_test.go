package bbl

import "testing"

func TestPersonValidation_WithName(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("person", "")
	fields := map[string]any{"name": "Jane Doe"}
	errs := validateRecord("public", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestPersonValidation_MissingName(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("person", "")
	fields := map[string]any{}
	errs := validateRecord("public", fields, defs)
	if !hasError(errs, "name", "not_empty") {
		t.Errorf("expected name not_empty error, got: %v", errs)
	}
}

func TestPersonValidation_EmptyName(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("person", "")
	fields := map[string]any{"name": ""}
	errs := validateRecord("public", fields, defs)
	if !hasError(errs, "name", "not_empty") {
		t.Errorf("expected name not_empty error, got: %v", errs)
	}
}
