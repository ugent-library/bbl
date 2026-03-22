package bbl

import (
	"fmt"

	"github.com/jackc/pgx/v5"
)

// assertion holds metadata for a single assertion row.
// Used both for pin computation and for noop/curator-lock checks.
type assertion struct {
	id             int64
	userID         *ID
	role           string
	sourceRecordID *ID
	source         string
	pinned         bool
	hidden         bool
}

// firstPinned returns the first pinned assertion, or nil.
// For exclusive pin, all pinned rows share the same asserter,
// so any pinned row is representative.
func firstPinned(assertions []assertion) *assertion {
	for i, a := range assertions {
		if a.pinned {
			return &assertions[i]
		}
	}
	return nil
}

// firstHuman returns the first human assertion (pinned or not), or nil.
func firstHuman(assertions []assertion) *assertion {
	for i, a := range assertions {
		if a.userID != nil {
			return &assertions[i]
		}
	}
	return nil
}

// resolveExclusivePin returns the desired pinned state for each assertion.
// Rule: if any human assertion exists, pin all human rows;
// else pin all rows from the highest-priority source.
func resolveExclusivePin(assertions []assertion, priorities map[string]int) []bool {
	result := make([]bool, len(assertions))

	hasHuman := false
	var winnerSourceRecordID *ID
	bestPriority := -1

	for _, a := range assertions {
		if a.userID != nil {
			hasHuman = true
			break
		}
		if a.sourceRecordID != nil {
			p := priorities[a.source]
			if p > bestPriority {
				bestPriority = p
				winnerSourceRecordID = a.sourceRecordID
			}
		}
	}

	for i, a := range assertions {
		if hasHuman {
			result[i] = a.userID != nil
		} else {
			result[i] = a.sourceRecordID != nil &&
				winnerSourceRecordID != nil &&
				*a.sourceRecordID == *winnerSourceRecordID
		}
	}

	return result
}

// queuePinUpdates computes the desired pin state and queues UPDATE statements
// into the batch for any rows that need to change.
func queuePinUpdates(batch *pgx.Batch, rt string, assertions []assertion, priorities map[string]int) {
	if len(assertions) == 0 {
		return
	}
	desired := resolveExclusivePin(assertions, priorities)
	table := assertionsTable(rt)
	for i, a := range assertions {
		if a.pinned != desired[i] {
			batch.Queue(fmt.Sprintf(
				`UPDATE %s SET pinned = $1 WHERE id = $2`, table),
				desired[i], a.id)
		}
	}
}
