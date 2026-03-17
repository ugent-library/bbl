package bbl

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// WorkProfiles holds the resolved work profiles, loaded once at startup.
// Kind order matches definition order in the YAML file.
type WorkProfiles struct {
	Kinds []WorkKind `yaml:"work_kinds"`
}

// WorkKind pairs a kind name with its profile.
type WorkKind struct {
	Name   string         `yaml:"name"`
	Fields []WorkFieldDef `yaml:"fields"`
}

// WorkFieldDef describes one field in a kind profile.
type WorkFieldDef struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"-"` // resolved from workFieldCatalog at load time
	Required bool     `yaml:"required,omitempty"`
	Schemes  []string `yaml:"schemes,omitempty"` // for identifier_list, classification_list
}

// workFieldCatalog is the canonical set of field names and their types.
// Every field name used in a profile YAML must exist here with a matching type.
// Relation fields (stored in separate tables) are included alongside scalar fields.
var workFieldCatalog = map[string]string{
	// Scalar fields
	"article_number":       "text",
	"book_title":           "text",
	"edition":              "text",
	"issue":                "text",
	"issue_title":          "text",
	"journal_abbreviation": "text",
	"journal_title":        "text",
	"place_of_publication": "text",
	"publication_status":   "text",
	"publication_year":     "year",
	"publisher":            "text",
	"report_number":        "text",
	"series_title":         "text",
	"total_pages":          "text",
	"volume":               "text",

	// Compound scalar fields (stored as JSON in bbl_work_assertions)
	"conference": "conference",
	"pages":      "extent",

	// Relation fields (stored in separate tables)
	"abstracts":       "text_list",
	"classifications": "classification_list",
	"contributors":    "contributor_list",
	"identifiers":     "identifier_list",
	"keywords":        "string_list",
	"lay_summaries":   "text_list",
	"notes":           "note_list",
	"titles":          "text_list",
}

// LoadWorkProfiles reads the YAML profile config and validates field names
// against the Go field catalog. Returns an error if any field name is unknown
// or has a mismatched type.
func LoadWorkProfiles(path string) (*WorkProfiles, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p WorkProfiles
	if err := yaml.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}
	if len(p.Kinds) == 0 {
		return nil, fmt.Errorf("parse profiles: no kinds defined")
	}
	for _, wk := range p.Kinds {
		if len(wk.Fields) == 0 {
			return nil, fmt.Errorf("profile %q: no fields defined", wk.Name)
		}
		for i := range wk.Fields {
			f := &wk.Fields[i]
			catalogType, ok := workFieldCatalog[f.Name]
			if !ok {
				return nil, fmt.Errorf("profile %q field %d: unknown field name %q", wk.Name, i, f.Name)
			}
			f.Type = catalogType
		}
	}
	return &p, nil
}

// Profile returns the kind definition, or nil if unknown.
func (p *WorkProfiles) Profile(kind string) *WorkKind {
	if p == nil {
		return nil
	}
	for i := range p.Kinds {
		if p.Kinds[i].Name == kind {
			return &p.Kinds[i]
		}
	}
	return nil
}

