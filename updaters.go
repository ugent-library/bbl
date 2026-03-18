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

// updater is the interface implemented by all update types.
// Unexported method names ensure only the bbl package can define updates.
type updater interface {
	// name returns "op:target" (e.g. "set:work_volume", "create:work").
	name() string

	// needs declares what state must be pre-fetched before apply.
	needs() updateNeeds

	// apply computes the effect of the update. Pure: no DB access.
	// Returns nil when the update is a noop (no change).
	apply(state updateState, userID *ID) (*updateEffect, error)

	// write executes the update's SQL against the transaction.
	// Called only for non-noop updates, after the bbl_revs row is inserted.
	// revID is the bigint identity of the current rev.
	write(ctx context.Context, tx pgx.Tx, revID int64) error
}

// updateEffect is what apply returns for non-noop updates.
type updateEffect struct {
	recordType string
	recordID   ID
	record     any // *Work, *Person, etc. — nil for field/relation updates
	// autoPin runs after all writes to evaluate pinning for the affected grouping key.
	// Receives ctx, tx, and source priorities. Nil means no auto-pin needed.
	autoPin func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error
}

// updateNeeds declares what existing state must be pre-fetched.
type updateNeeds struct {
	organizationIDs []ID
	personIDs       []ID
	projectIDs      []ID
	workIDs         []ID
}

// updateState holds pre-fetched entity state for a batch of updates.
type updateState struct {
	organizations map[ID]*Organization
	people        map[ID]*Person
	projects      map[ID]*Project
	works         map[ID]*Work
	priorities    map[string]int // source priorities for auto-pin
}
