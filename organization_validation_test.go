package bbl

import "testing"

func TestOrgValidation_PublicWithNames(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("organization", "department")
	fields := map[string]any{"names": []Text{{Lang: "eng", Val: "Physics"}}}
	errs := validateRecord("public", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestOrgValidation_PublicMissingNames(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("organization", "department")
	fields := map[string]any{}
	errs := validateRecord("public", fields, defs)
	if !hasError(errs, "names", "not_empty") {
		t.Errorf("expected names not_empty error, got: %v", errs)
	}
}

func TestOrgValidation_DeletedNoNames(t *testing.T) {
	p := testProfiles(t)
	defs := p.FieldDefs("organization", "department")
	fields := map[string]any{}
	errs := validateRecord("deleted", fields, defs)
	if errs != nil {
		t.Errorf("expected no errors for deleted org, got: %v", errs)
	}
}
