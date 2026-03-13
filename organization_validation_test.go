package bbl

import "testing"

func TestValidateOrganization_ValidPublic(t *testing.T) {
	o := &Organization{
		Kind:   "department",
		Status: OrganizationStatusPublic,
		Names: []Text{{Lang: "eng", Val: "Physics"}},
	}
	errs := ValidateOrganization(o)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateOrganization_InvalidStatus(t *testing.T) {
	o := &Organization{
		Kind:   "department",
		Status: "bogus",
	}
	errs := ValidateOrganization(o)
	if !hasError(errs, "status", "one_of") {
		t.Errorf("expected status one_of error, got: %v", errs)
	}
}

func TestValidateOrganization_MissingKind(t *testing.T) {
	o := &Organization{
		Status: OrganizationStatusPublic,
		Names:  []Text{{Lang: "eng", Val: "Physics"}},
	}
	errs := ValidateOrganization(o)
	if !hasError(errs, "kind", "not_blank") {
		t.Errorf("expected kind not_blank error, got: %v", errs)
	}
}

func TestValidateOrganization_InvalidLang(t *testing.T) {
	o := &Organization{
		Kind:   "department",
		Status: OrganizationStatusPublic,
		Names:  []Text{{Lang: "xx", Val: "Physics"}},
	}
	errs := ValidateOrganization(o)
	if !hasError(errs, "names[0].lang", "iso639_2") {
		t.Errorf("expected lang error, got: %v", errs)
	}
}

func TestValidateOrganization_MissingNamesWhenPublic(t *testing.T) {
	o := &Organization{Kind: "department", Status: OrganizationStatusPublic}
	errs := ValidateOrganization(o)
	if !hasError(errs, "names", "not_empty") {
		t.Errorf("expected names not_empty error, got: %v", errs)
	}
}

func TestValidateOrganization_MissingNamesWhenDeleted(t *testing.T) {
	o := &Organization{Kind: "department", Status: OrganizationStatusDeleted}
	errs := ValidateOrganization(o)
	if errs != nil {
		t.Errorf("expected no errors for deleted org, got: %v", errs)
	}
}
