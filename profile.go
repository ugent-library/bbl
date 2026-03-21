package bbl

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FieldDef describes one field in a profile.
type FieldDef struct {
	ft       *fieldType
	Name     string
	Type     string   // resolved fieldType name (for views)
	Required string   // "", "always", "public"
	Schemes  []string // for identifier, classification
}

// IsRequired reports whether the field has any required constraint.
func (f FieldDef) IsRequired() bool { return f.Required != "" }

// Profiles holds resolved profiles for all entity types, loaded once at startup.
type Profiles struct {
	Work         map[string][]FieldDef
	Organization map[string][]FieldDef
	Person       []FieldDef
	Project      []FieldDef
	workKinds    []string // ordered from YAML
	orgKinds     []string
}

// WorkKinds returns work kind names in definition order.
func (p *Profiles) WorkKinds() []string { return p.workKinds }

// OrganizationKinds returns organization kind names in definition order.
func (p *Profiles) OrganizationKinds() []string { return p.orgKinds }

// FieldDefs returns the field definitions for a record type and kind.
// Returns nil if the record type or kind is unknown.
func (p *Profiles) FieldDefs(recordType, kind string) []FieldDef {
	if p == nil {
		return nil
	}
	switch recordType {
	case "work":
		return p.Work[kind]
	case "person":
		return p.Person
	case "project":
		return p.Project
	case "organization":
		return p.Organization[kind]
	}
	return nil
}

// --- YAML loading ---

type profilesFile struct {
	WorkKinds []profileKind `yaml:"work_kinds"`
	OrgKinds  []profileKind `yaml:"organization_kinds"`
	Person    profileKind   `yaml:"person"`
	Project   profileKind   `yaml:"project"`
}

type profileKind struct {
	Name   string            `yaml:"name,omitempty"`
	Fields []profileFieldDef `yaml:"fields"`
}

type profileFieldDef struct {
	Name     string   `yaml:"name"`
	Required string   `yaml:"required,omitempty"`
	Schemes  []string `yaml:"schemes,omitempty"`
}

// LoadProfiles reads the YAML profile config and validates field names
// against the field type registry. Returns an error if any field name
// is unknown or any section is missing.
func LoadProfiles(path string) (*Profiles, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f profilesFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}

	p := &Profiles{
		Work:         make(map[string][]FieldDef),
		Organization: make(map[string][]FieldDef),
	}

	// Work kinds.
	if len(f.WorkKinds) == 0 {
		return nil, fmt.Errorf("parse profiles: no work kinds defined")
	}
	for _, wk := range f.WorkKinds {
		defs, err := resolveFieldDefs("work", wk.Name, wk.Fields)
		if err != nil {
			return nil, err
		}
		p.Work[wk.Name] = defs
		p.workKinds = append(p.workKinds, wk.Name)
	}

	// Organization kinds.
	if len(f.OrgKinds) == 0 {
		return nil, fmt.Errorf("parse profiles: no organization kinds defined")
	}
	for _, ok := range f.OrgKinds {
		defs, err := resolveFieldDefs("organization", ok.Name, ok.Fields)
		if err != nil {
			return nil, err
		}
		p.Organization[ok.Name] = defs
		p.orgKinds = append(p.orgKinds, ok.Name)
	}

	// Person.
	if len(f.Person.Fields) == 0 {
		return nil, fmt.Errorf("parse profiles: no person fields defined")
	}
	defs, err := resolveFieldDefs("person", "person", f.Person.Fields)
	if err != nil {
		return nil, err
	}
	p.Person = defs

	// Project.
	if len(f.Project.Fields) == 0 {
		return nil, fmt.Errorf("parse profiles: no project fields defined")
	}
	defs, err = resolveFieldDefs("project", "project", f.Project.Fields)
	if err != nil {
		return nil, err
	}
	p.Project = defs

	return p, nil
}

func resolveFieldDefs(entityType, profileName string, fields []profileFieldDef) ([]FieldDef, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("profile %q: no fields defined", profileName)
	}
	fieldTypes, ok := entityFieldTypes[entityType]
	if !ok {
		return nil, fmt.Errorf("profile %q: unknown entity type %q", profileName, entityType)
	}
	defs := make([]FieldDef, len(fields))
	for i, f := range fields {
		ftName, ok := fieldTypes[f.Name]
		if !ok {
			return nil, fmt.Errorf("profile %q field %d: unknown field name %q", profileName, i, f.Name)
		}
		ft, ok := fieldTypeRegistry[ftName]
		if !ok {
			return nil, fmt.Errorf("profile %q field %d: unknown field type %q for field %q", profileName, i, ftName, f.Name)
		}
		if f.Required != "" && f.Required != "always" && f.Required != "public" {
			return nil, fmt.Errorf("profile %q field %q: invalid required value %q (must be empty, \"always\", or \"public\")", profileName, f.Name, f.Required)
		}
		defs[i] = FieldDef{
			ft:       ft,
			Name:     f.Name,
			Type:     ftName,
			Required: f.Required,
			Schemes:  f.Schemes,
		}
	}
	return defs, nil
}
