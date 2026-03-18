package bbl

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// Record type discriminators for entity types.
const (
	RecordTypeOrganization = "organization"
	RecordTypePerson       = "person"
	RecordTypeProject      = "project"
	RecordTypeWork         = "work"
)

// mutation is the interface implemented by all mutation types.
// Unexported method names ensure only the bbl package can define mutations.
type mutation interface {
	// mutationName returns a human-readable name for the mutation.
	mutationName() string

	// needs declares what state must be pre-fetched before apply.
	needs() mutationNeeds

	// apply computes the effect of the mutation. Pure: no DB access.
	// Returns nil when the mutation is a noop (no change).
	apply(state mutationState, userID *ID) (*mutationEffect, error)

	// write executes the mutation's SQL against the transaction.
	// Called only for non-noop mutations, after the bbl_revs row is inserted.
	// revID is the bigint identity of the current rev.
	write(ctx context.Context, tx pgx.Tx, revID int64) error
}

// mutationEffect is what apply returns for non-noop mutations.
type mutationEffect struct {
	recordType string
	recordID   ID
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
