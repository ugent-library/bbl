package bbl

import (
	"context"
	"iter"
	"time"
)

const (
	ProjectStatusPublic  = "public"
	ProjectStatusDeleted = "deleted"
)

type Project struct {
	ID          ID
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedByID *ID
	UpdatedByID *ID
	Status      string
	StartDate   *time.Time
	EndDate     *time.Time
	DeletedAt   *time.Time
	DeletedByID *ID

	// Populated from the cache column on read.
	Titles       []Title         `json:"titles,omitempty"`
	Descriptions []Text          `json:"descriptions,omitempty"`
	Identifiers  []Identifier    `json:"identifiers,omitempty"`
	Participants []ProjectParticipant `json:"participants,omitempty"`
}

// ProjectParticipant is a participant read from the cache column.
type ProjectParticipant struct {
	PersonID ID     `json:"person_id"`
	Role     string `json:"role,omitempty"`
}

// ImportProjectInput carries all data for one project record arriving from a source.
type ImportProjectInput struct {
	ID        *ID        `json:"id,omitempty"` // if set, reuse this ID (legacy import); otherwise generate
	SourceID  string     `json:"source_id"`
	Status    string     `json:"status,omitempty"` // if empty, defaults to ProjectStatusPublic
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	// Text list fields.
	Titles       []Title `json:"titles,omitempty"`
	Descriptions []Text  `json:"descriptions,omitempty"`

	Participants []ImportProjectParticipant `json:"participants,omitempty"`

	// SourceRecord is the original payload from the source (XML, JSON, etc.).
	SourceRecord []byte `json:"-"`
}

// ImportProjectParticipant links a project to a person during import.
type ImportProjectParticipant struct {
	Ref  Ref    `json:"ref"`
	Role string `json:"role,omitempty"`
}

// ProjectSource is the interface implemented by project import sources.
type ProjectSource interface {
	Iter(ctx context.Context) (iter.Seq2[*ImportProjectInput, error], error)
}
