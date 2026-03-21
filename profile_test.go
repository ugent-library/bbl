package bbl

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWorkFieldTypes(t *testing.T) {
	// Verify the field type map has entries and each has a non-empty type.
	if len(workFieldTypes) == 0 {
		t.Fatal("workFieldTypes is empty")
	}
	for name, typ := range workFieldTypes {
		if name == "" {
			t.Error("empty field name in workFieldTypes")
		}
		if typ == "" {
			t.Errorf("field %q has empty type", name)
		}
		if _, ok := fieldTypeRegistry[typ]; !ok {
			t.Errorf("field %q: type %q not in fieldTypeRegistry", name, typ)
		}
	}
}

func TestLoadProfiles(t *testing.T) {
	p, err := LoadProfiles("testdata/profiles.yaml")
	if err != nil {
		t.Fatalf("LoadProfiles: %v", err)
	}

	// Two work kinds defined.
	if got := len(p.Work); got != 2 {
		t.Fatalf("expected 2 work kinds, got %d", got)
	}
	if p.WorkKinds()[0] != "journal_article" {
		t.Errorf("first kind = %q, want journal_article", p.WorkKinds()[0])
	}
	if p.WorkKinds()[1] != "book" {
		t.Errorf("second kind = %q, want book", p.WorkKinds()[1])
	}

	// journal_article field count and order preserved.
	ja := p.FieldDefs("work", "journal_article")
	if ja == nil {
		t.Fatal("journal_article profile is nil")
	}
	if got := len(ja); got != 9 {
		t.Fatalf("journal_article: expected 9 fields, got %d", got)
	}
	if ja[0].Name != "titles" {
		t.Errorf("first field = %q, want titles", ja[0].Name)
	}
	if ja[0].Required != "always" {
		t.Error("titles should be required: always")
	}
	// Type resolved from workFieldTypes.
	if ja[0].Type != "title" {
		t.Errorf("titles type = %q, want title", ja[0].Type)
	}
	// identifiers schemes.
	idField := ja[2]
	if idField.Name != "identifiers" {
		t.Fatalf("field 2 = %q, want identifiers", idField.Name)
	}
	if want := []string{"doi", "issn"}; !reflect.DeepEqual(idField.Schemes, want) {
		t.Errorf("identifiers schemes = %v, want %v", idField.Schemes, want)
	}

	// Unknown kind returns nil.
	if p.FieldDefs("work", "nonexistent") != nil {
		t.Error("expected nil for unknown kind")
	}

	// Person fields.
	if p.Person == nil {
		t.Fatal("person fields are nil")
	}
	if p.Person[0].Name != "name" {
		t.Errorf("first person field = %q, want name", p.Person[0].Name)
	}

	// Project fields.
	if p.Project == nil {
		t.Fatal("project fields are nil")
	}

	// Organization kinds.
	if len(p.Organization) == 0 {
		t.Fatal("no organization kinds")
	}
}

func TestLoadProfilesValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "unknown field",
			yaml:    "work_kinds:\n  - name: test\n    fields:\n      - name: bogus\nperson:\n  fields:\n    - name: name\nproject:\n  fields:\n    - name: titles\norganization_kinds:\n  - name: dept\n    fields:\n      - name: names\n",
			wantErr: "unknown field name",
		},
		{
			name:    "no work kinds",
			yaml:    "work_kinds: []\nperson:\n  fields:\n    - name: name\nproject:\n  fields:\n    - name: titles\norganization_kinds:\n  - name: dept\n    fields:\n      - name: names\n",
			wantErr: "no work kinds defined",
		},
		{
			name:    "no fields",
			yaml:    "work_kinds:\n  - name: test\n    fields: []\nperson:\n  fields:\n    - name: name\nproject:\n  fields:\n    - name: titles\norganization_kinds:\n  - name: dept\n    fields:\n      - name: names\n",
			wantErr: "no fields defined",
		},
		{
			name:    "invalid required",
			yaml:    "work_kinds:\n  - name: test\n    fields:\n      - name: titles\n        required: bogus\nperson:\n  fields:\n    - name: name\nproject:\n  fields:\n    - name: titles\norganization_kinds:\n  - name: dept\n    fields:\n      - name: names\n",
			wantErr: "invalid required value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "profile-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(tt.yaml); err != nil {
				t.Fatal(err)
			}
			f.Close()

			_, err = LoadProfiles(f.Name())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
