package bbl

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Record type discriminators used in bbl_mutations.entity_type.
const (
	RecordTypeOrganization = "organization"
	RecordTypePerson       = "person"
	RecordTypeProject      = "project"
	RecordTypeWork         = "work"
)

// Op type discriminators used in bbl_mutations.op_type.
const (
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
)

// Diff is the audit record stored in bbl_mutations.diff.
// Args holds the new field values written by the mutation.
// Prev holds the prior values of those same fields (omitted for creates).
type Diff struct {
	Args any `json:"args,omitempty"`
	Prev any `json:"prev,omitempty"`
}

// mutation is the interface implemented by all mutation types.
// Unexported method names ensure only the bbl package can define mutations.
type mutation interface {
	// mutationName returns the audit name stored in bbl_mutations.name.
	mutationName() string

	// needs declares what state must be pre-fetched before apply.
	needs() mutationNeeds

	// apply computes the effect of the mutation. Pure: no DB access.
	// Returns nil when the mutation is a noop (no change).
	apply(state mutationState, userID *ID) (*mutationEffect, error)

	// write executes the mutation's SQL against the transaction.
	// Called only for non-noop mutations, after the bbl_revs row is inserted.
	write(ctx context.Context, tx pgx.Tx) error
}

// mutationEffect is what apply returns for non-noop mutations.
type mutationEffect struct {
	recordType string
	recordID   ID
	opType     string
	diff       Diff
	record     any // *Work, *Person, etc. — nil for field/relation mutations
	// autoPin runs after all writes to evaluate pinning for the affected grouping key.
	// Receives ctx, tx, and source priorities. Nil means no auto-pin needed.
	autoPin func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error
}

// mutationNeeds declares what existing state must be pre-fetched.
type mutationNeeds struct {
	organizationIDs []ID
	personIDs       []ID
	projectIDs      []ID
	workIDs         []ID
}

// mutationState holds pre-fetched entity state for a batch of mutations.
type mutationState struct {
	organizations map[ID]*Organization
	people        map[ID]*Person
	projects      map[ID]*Project
	works         map[ID]*Work
	priorities    map[string]int // source priorities for auto-pin
}
