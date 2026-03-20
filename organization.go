package bbl

import (
	"context"
	"iter"
	"time"
)

const (
	OrganizationStatusPublic  = "public"
	OrganizationStatusDeleted = "deleted"
)

type Organization struct {
	ID          ID
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedByID *ID
	UpdatedByID *ID
	Kind        string
	Status      string
	StartDate   *time.Time
	EndDate     *time.Time
	DeletedAt   *time.Time
	DeletedByID *ID

	// Relations — populated from the cache column on read.
	Identifiers []Identifier      `json:"identifiers,omitempty"`
	Names       []Text            `json:"names,omitempty"`
	Rels        []OrganizationRel `json:"rels,omitempty"`
}

// OrganizationRel links two organizations with a typed, optionally temporal relationship.
type OrganizationRel struct {
	RelOrganizationID ID         `json:"rel_organization_id"`
	Kind              string     `json:"kind"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
}

// ImportOrganizationInput carries all data for one organization record arriving from a source.
type ImportOrganizationInput struct {
	ID        *ID        `json:"id,omitempty"`
	SourceID  string     `json:"source_id"`
	Kind      string     `json:"kind"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	// Org names are language-keyed relations (bbl_organization_names).
	Names []Text `json:"names,omitempty"`

	Identifiers []Identifier            `json:"identifiers,omitempty"`
	Rels        []ImportOrganizationRel `json:"rels,omitempty"`

	// SourceRecord is the original payload from the source (XML, JSON, etc.).
	SourceRecord []byte `json:"-"`
}

// ImportOrganizationRel describes a relationship to another organization.
type ImportOrganizationRel struct {
	Ref       Ref        `json:"ref"`
	Kind      string     `json:"kind"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// OrganizationSource is the interface implemented by organization import sources.
type OrganizationSource interface {
	Iter(ctx context.Context) (iter.Seq2[*ImportOrganizationInput, error], error)
}
