package bbl

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestWorkFieldCatalog(t *testing.T) {
	// Verify the catalog has entries and each has a non-empty type.
	if len(workFieldCatalog) == 0 {
		t.Fatal("workFieldCatalog is empty")
	}
	for name, typ := range workFieldCatalog {
		if name == "" {
			t.Error("empty field name in catalog")
		}
		if typ == "" {
			t.Errorf("field %q has empty type", name)
		}
	}
}

func TestLoadWorkProfiles(t *testing.T) {
	p, err := LoadWorkProfiles("testdata/profiles.yaml")
	if err != nil {
		t.Fatalf("LoadWorkProfiles: %v", err)
	}

	// Two kinds defined, in YAML order.
	if got := len(p.Kinds); got != 2 {
		t.Fatalf("expected 2 kinds, got %d", got)
	}
	if p.Kinds[0].Name != "journal_article" {
		t.Errorf("first kind = %q, want journal_article", p.Kinds[0].Name)
	}
	if p.Kinds[1].Name != "book" {
		t.Errorf("second kind = %q, want book", p.Kinds[1].Name)
	}

	// journal_article field count and order preserved.
	ja := p.Profile("journal_article")
	if ja == nil {
		t.Fatal("journal_article profile is nil")
	}
	if got := len(ja.Fields); got != 9 {
		t.Fatalf("journal_article: expected 9 fields, got %d", got)
	}
	if ja.Fields[0].Name != "titles" {
		t.Errorf("first field = %q, want titles", ja.Fields[0].Name)
	}
	if !ja.Fields[0].Required {
		t.Error("titles should be required")
	}
	// Type resolved from catalog.
	if ja.Fields[0].Type != "text_list" {
		t.Errorf("titles type = %q, want text_list", ja.Fields[0].Type)
	}
	// identifiers schemes.
	idField := ja.Fields[2]
	if idField.Name != "identifiers" {
		t.Fatalf("field 2 = %q, want identifiers", idField.Name)
	}
	if want := []string{"doi", "issn"}; !reflect.DeepEqual(idField.Schemes, want) {
		t.Errorf("identifiers schemes = %v, want %v", idField.Schemes, want)
	}

	// Unknown kind returns nil.
	if p.Profile("nonexistent") != nil {
		t.Error("expected nil for unknown kind")
	}
}

func TestLoadWorkProfilesValidation(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "unknown field",
			yaml:    "work_kinds:\n  - name: test\n    fields:\n      - name: bogus\n",
			wantErr: "unknown field name",
		},
		{
			name:    "no kinds",
			yaml:    "work_kinds: []\n",
			wantErr: "no kinds defined",
		},
		{
			name:    "no fields",
			yaml:    "work_kinds:\n  - name: test\n    fields: []\n",
			wantErr: "no fields defined",
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

			_, err = LoadWorkProfiles(f.Name())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}
