package bbl

import "time"

// Text is a language-tagged string value (title, abstract, etc.).
type Text struct {
	Lang string `json:"lang"`
	Val  string `json:"val"`
}

// Title is a language-tagged title (for projects, works, etc.).
// Separate from Text to allow future expansion (e.g. kind: translated_title).
type Title struct {
	Lang string `json:"lang"`
	Val  string `json:"val"`
}

// Conference is a conference associated with a work.
type Conference struct {
	Name      string    `json:"name,omitempty"`
	Organizer string    `json:"organizer,omitempty"`
	Location  string    `json:"location,omitempty"`
	StartDate time.Time `json:"start_date,omitzero"`
	EndDate   time.Time `json:"end_date,omitzero"`
}

// Extent is a range (e.g. pages).
type Extent struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// Note is a typed annotation on a work.
type Note struct {
	Kind string `json:"kind,omitempty"`
	Val  string `json:"val"`
}

// Keyword is a keyword/subject term on a work.
type Keyword struct {
	Val string `json:"val"`
}

// Identifier is a scheme/val pair used for entity identifiers across all entity types.
type Identifier struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
}

// Ref identifies an entity for cross-entity linking during import.
// Exactly one field should be set.
type Ref struct {
	ID         *ID         `json:"id,omitempty"`
	SourceID   string      `json:"source_id,omitempty"`
	Identifier *Identifier `json:"identifier,omitempty"`
}
