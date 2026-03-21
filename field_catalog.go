package bbl

import "fmt"

// workFieldTypes maps work field names to their fieldType names.
// This is the Go-side mapping — the profile YAML declares which fields
// are active per kind, this provides the type for each field name.
var workFieldTypes = map[string]string{
	// Scalars
	"article_number":       "string",
	"book_title":           "string",
	"edition":              "string",
	"issue":                "string",
	"issue_title":          "string",
	"journal_abbreviation": "string",
	"journal_title":        "string",
	"place_of_publication": "string",
	"publication_status":   "string",
	"publication_year":     "string",
	"publisher":            "string",
	"report_number":        "string",
	"series_title":         "string",
	"total_pages":          "string",
	"volume":               "string",

	// Compound scalars
	"conference": "conference",
	"pages":      "extent",

	// Pure-value collections
	"abstracts":       "text",
	"classifications": "classification",
	"identifiers":     "identifier",
	"keywords":        "keyword",
	"lay_summaries":   "text",
	"notes":           "note",
	"titles":          "title",

	// FK-bearing collections
	"contributors":  "workContributor",
	"projects":      "workProject",
	"organizations": "workOrganization",
	"rels":          "workRel",
}

// personFieldTypes maps person field names to their fieldType names.
var personFieldTypes = map[string]string{
	"name":         "string",
	"given_name":   "string",
	"middle_name":  "string",
	"family_name":  "string",
	"identifiers":  "identifier",
	"affiliations": "personAffiliation",
}

// projectFieldTypes maps project field names to their fieldType names.
var projectFieldTypes = map[string]string{
	"titles":       "title",
	"descriptions": "text",
	"identifiers":  "identifier",
	"participants": "projectParticipant",
}

// organizationFieldTypes maps organization field names to their fieldType names.
var organizationFieldTypes = map[string]string{
	"identifiers": "identifier",
	"names":       "text",
	"rels":        "organizationRel",
}

// entityFieldTypes maps entity type → field name → fieldType name.
var entityFieldTypes = map[string]map[string]string{
	"work":         workFieldTypes,
	"person":       personFieldTypes,
	"project":      projectFieldTypes,
	"organization": organizationFieldTypes,
}

// resolveFieldType looks up the fieldType for a given entity type and field name.
func resolveFieldType(entityType, field string) (*fieldType, error) {
	fieldTypes, ok := entityFieldTypes[entityType]
	if !ok {
		return nil, fmt.Errorf("unknown entity type %q", entityType)
	}
	ftName, ok := fieldTypes[field]
	if !ok {
		return nil, fmt.Errorf("unknown field %q for entity %q", field, entityType)
	}
	ft, ok := fieldTypeRegistry[ftName]
	if !ok {
		return nil, fmt.Errorf("unknown field type %q for field %q", ftName, field)
	}
	return ft, nil
}
