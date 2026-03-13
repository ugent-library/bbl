package bbl

import "testing"

func TestValidateProject_ValidPublic(t *testing.T) {
	p := &Project{Status: ProjectStatusPublic, Titles: []Title{{Val: "My Project"}}}
	errs := ValidateProject(p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateProject_InvalidStatus(t *testing.T) {
	p := &Project{Status: "bogus"}
	errs := ValidateProject(p)
	if !hasError(errs, "status", "one_of") {
		t.Errorf("expected status one_of error, got: %v", errs)
	}
}

func TestValidateProject_MissingTitles(t *testing.T) {
	p := &Project{Status: ProjectStatusPublic}
	errs := ValidateProject(p)
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error, got: %v", errs)
	}
}

func TestValidateProject_MissingTitlesWhenDeleted(t *testing.T) {
	p := &Project{Status: ProjectStatusDeleted}
	errs := ValidateProject(p)
	if !hasError(errs, "titles", "not_empty") {
		t.Errorf("expected titles not_empty error, got: %v", errs)
	}
}

func TestValidateProject_ValidDeleted(t *testing.T) {
	p := &Project{Status: ProjectStatusDeleted, Titles: []Title{{Val: "My Project"}}}
	errs := ValidateProject(p)
	if errs != nil {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateProject_BlankTitleVal(t *testing.T) {
	p := &Project{Status: ProjectStatusPublic, Titles: []Title{{Lang: "en", Val: ""}}}
	errs := ValidateProject(p)
	if !hasError(errs, "titles[0].val", "not_blank") {
		t.Errorf("expected titles[0].val not_blank error, got: %v", errs)
	}
}
