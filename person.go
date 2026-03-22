package bbl

import (
	"context"
	"iter"
	"time"
)

const (
	PersonStatusPublic  = "public"
	PersonStatusDeleted = "deleted"
)

type Person struct {
	ID          ID
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedByID *ID
	UpdatedByID *ID
	Status      string
	DeletedAt   *time.Time
	DeletedByID *ID

	// Populated from the cache column on read.
	Name          string               `json:"name"`
	GivenName     string               `json:"given_name,omitempty"`
	MiddleName    string               `json:"middle_name,omitempty"`
	FamilyName    string               `json:"family_name,omitempty"`
	Identifiers   []Identifier         `json:"identifiers,omitempty"`
	Affiliations []PersonAffiliation `json:"affiliations,omitempty"`
}

// PersonAffiliation is an affiliation read from the cache column.
type PersonAffiliation struct {
	OrganizationID ID `json:"organization_id"`
}

// ImportPersonInput carries all data for one person record arriving from a source.
type ImportPersonInput struct {
	ID       *ID    `json:"id,omitempty"`
	SourceID string `json:"source_id"`

	// Scalar fields.
	Name       string `json:"name,omitempty"`
	GivenName  string `json:"given_name,omitempty"`
	MiddleName string `json:"middle_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`

	Identifiers  []Identifier              `json:"identifiers,omitempty"`
	Affiliations []ImportPersonAffiliation `json:"affiliations,omitempty"`

	// SourceRecord is the original payload from the source (XML, JSON, etc.).
	SourceRecord []byte `json:"-"`
}

// ImportPersonAffiliation links a person to an organization during import.
type ImportPersonAffiliation struct {
	Ref Ref `json:"ref"`
}

// PersonSource is the interface implemented by person import sources.
type PersonSource interface {
	Iter(ctx context.Context) (iter.Seq2[*ImportPersonInput, error], error)
}
