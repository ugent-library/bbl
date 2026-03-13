package bbl

import "testing"

func TestValidatePerson_ValidPublic(t *testing.T) {
	p := &Person{Status: PersonStatusPublic, Name: "Jane Doe"}
	errs := ValidatePerson(p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidatePerson_InvalidStatus(t *testing.T) {
	p := &Person{Status: "bogus"}
	errs := ValidatePerson(p)
	if !hasError(errs, "status", "one_of") {
		t.Errorf("expected status one_of error, got: %v", errs)
	}
}

func TestValidatePerson_MissingNameWhenPublic(t *testing.T) {
	p := &Person{Status: PersonStatusPublic}
	errs := ValidatePerson(p)
	if !hasError(errs, "name", "not_blank") {
		t.Errorf("expected name not_blank error, got: %v", errs)
	}
}

func TestValidatePerson_MissingNameWhenDeleted(t *testing.T) {
	p := &Person{Status: PersonStatusDeleted}
	errs := ValidatePerson(p)
	if !hasError(errs, "name", "not_blank") {
		t.Errorf("expected name not_blank error, got: %v", errs)
	}
}

func TestValidatePerson_ValidDeleted(t *testing.T) {
	p := &Person{Status: PersonStatusDeleted, Name: "Jane Doe"}
	errs := ValidatePerson(p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}
