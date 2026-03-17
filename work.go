package bbl

import (
	"context"
	"iter"
	"time"
)

// Work status values.
const (
	WorkStatusPrivate = "private"
	WorkStatusPublic  = "public"
	WorkStatusDeleted = "deleted"
)

// Work review status values. Empty string means not in review.
const (
	WorkReviewPending  = "pending"
	WorkReviewInReview = "in_review"
	WorkReviewReturned = "returned"
)

// Work delete kind values (set when status = deleted).
const (
	WorkDeleteWithdrawn = "withdrawn"
	WorkDeleteRetracted = "retracted"
	WorkDeleteTakedown  = "takedown"
)

type Work struct {
	ID           ID
	Version      int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedByID  *ID
	UpdatedByID  *ID
	Kind         string // denormalized from pinned kind assertion
	Status       string
	ReviewStatus string // empty = not in review
	DeleteKind   string
	DeletedAt    *time.Time
	DeletedByID  *ID

	// Scalar fields — populated from str_fields in the cache column on read.
	ArticleNumber       string     `json:"article_number,omitempty"`
	BookTitle           string     `json:"book_title,omitempty"`
	Conference          Conference `json:"conference,omitzero"`
	Edition             string     `json:"edition,omitempty"`
	Issue               string     `json:"issue,omitempty"`
	IssueTitle          string     `json:"issue_title,omitempty"`
	JournalAbbreviation string     `json:"journal_abbreviation,omitempty"`
	JournalTitle        string     `json:"journal_title,omitempty"`
	Pages               Extent     `json:"pages,omitzero"`
	PlaceOfPublication  string     `json:"place_of_publication,omitempty"`
	PublicationStatus   string     `json:"publication_status,omitempty"`
	PublicationYear     string     `json:"publication_year,omitempty"`
	Publisher           string     `json:"publisher,omitempty"`
	ReportNumber        string     `json:"report_number,omitempty"`
	SeriesTitle         string     `json:"series_title,omitempty"`
	TotalPages          string     `json:"total_pages,omitempty"`
	Volume              string     `json:"volume,omitempty"`

	// Relations — populated from the cache column on read.
	Identifiers     []WorkIdentifier     `json:"identifiers,omitempty"`
	Classifications []WorkClassification `json:"classifications,omitempty"`
	Contributors    []WorkContributor    `json:"contributors,omitempty"`
	Titles          []Title              `json:"titles,omitempty"`
	Abstracts       []Text               `json:"abstracts,omitempty"`
	LaySummaries    []Text               `json:"lay_summaries,omitempty"`
	Notes           []Note               `json:"notes,omitempty"`
	Keywords        []Keyword            `json:"keywords,omitempty"`
}

// ImportWorkInput carries all data for one work record arriving from a source.
type ImportWorkInput struct {
	ID       *ID    `json:"id,omitempty"` // if set, reuse this ID (legacy import); otherwise generate
	SourceID string `json:"source_id"`
	Kind     string `json:"kind"`
	Status   string `json:"status,omitempty"`

	// Scalar fields.
	ArticleNumber       string          `json:"article_number,omitempty"`
	BookTitle           string          `json:"book_title,omitempty"`
	Conference          Conference `json:"conference,omitzero"`
	Edition             string          `json:"edition,omitempty"`
	Issue               string          `json:"issue,omitempty"`
	IssueTitle          string          `json:"issue_title,omitempty"`
	JournalAbbreviation string          `json:"journal_abbreviation,omitempty"`
	JournalTitle        string          `json:"journal_title,omitempty"`
	Pages               Extent     `json:"pages,omitzero"`
	PlaceOfPublication  string          `json:"place_of_publication,omitempty"`
	PublicationStatus   string          `json:"publication_status,omitempty"`
	PublicationYear     string          `json:"publication_year,omitempty"`
	Publisher           string          `json:"publisher,omitempty"`
	ReportNumber        string          `json:"report_number,omitempty"`
	SeriesTitle         string          `json:"series_title,omitempty"`
	TotalPages          string          `json:"total_pages,omitempty"`
	Volume              string          `json:"volume,omitempty"`

	// Relation data provided by the source.
	Identifiers     []Identifier             `json:"identifiers,omitempty"`
	Classifications []Identifier             `json:"classifications,omitempty"`
	Contributors    []ImportWorkContributor  `json:"contributors,omitempty"`
	Projects        []ImportWorkProject      `json:"projects,omitempty"`
	Organizations   []ImportWorkOrganization `json:"organizations,omitempty"`
	RelatedWorks    []ImportWorkRel          `json:"related_works,omitempty"`
	Titles          []Title                  `json:"titles,omitempty"`
	Abstracts       []Text                   `json:"abstracts,omitempty"`
	LaySummaries    []Text                   `json:"lay_summaries,omitempty"`
	Notes           []Note                   `json:"notes,omitempty"`
	Keywords        []string                 `json:"keywords,omitempty"`

	// SourceRecord is the original payload from the source (XML, JSON, etc.).
	// Stored as-is in bbl_work_sources for debugging and comparison.
	SourceRecord []byte `json:"-"`
}

// ImportWorkContributor is a contributor arriving from a source.
type ImportWorkContributor struct {
	PersonRef  *Ref     `json:"person_ref,omitempty"`
	Kind       string   `json:"kind,omitempty"` // "person" (default) or "organization"
	Roles      []string `json:"roles,omitempty"`
	Name       string   `json:"name,omitempty"`
	GivenName  string   `json:"given_name,omitempty"`
	MiddleName string   `json:"middle_name,omitempty"`
	FamilyName string   `json:"family_name,omitempty"`
}

// ImportWorkProject links a work to a project during import.
type ImportWorkProject struct {
	Ref Ref `json:"ref"`
}

// ImportWorkOrganization links a work to an organization during import.
type ImportWorkOrganization struct {
	Ref Ref `json:"ref"`
}

// ImportWorkRel links a work to a related work during import.
type ImportWorkRel struct {
	Ref  Ref    `json:"ref"`
	Kind string `json:"kind"`
}

// WorkIdentifier is a scheme/val pair read from the cache column.
type WorkIdentifier struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
	Source string `json:"source,omitempty"`
}

// WorkClassification is a scheme/val classification read from the cache column.
type WorkClassification struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
	Source string `json:"source,omitempty"`
}

// WorkContributor is a contributor read from the cache column.
type WorkContributor struct {
	Position   int      `json:"position"`
	Kind       string   `json:"kind,omitempty"`        // "person" (default) or "organization"
	PersonID   *ID      `json:"person_id,omitempty"`
	Name       string   `json:"name,omitempty"`
	GivenName  string   `json:"given_name,omitempty"`
	FamilyName string   `json:"family_name,omitempty"`
	Roles      []string `json:"roles,omitempty"`
}

// WorkSourceIter is implemented by sources that can iterate all records.
type WorkSourceIter interface {
	Iter(ctx context.Context) (iter.Seq2[*ImportWorkInput, error], error)
}

// WorkSourceGetter is implemented by sources that can fetch a single record by ID.
type WorkSourceGetter interface {
	Get(ctx context.Context, id string) (*ImportWorkInput, error)
}
