package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// rolePriority returns the pinning priority for a role.
// Higher = wins. Curator beats user.
func rolePriority(role string) int {
	switch role {
	case "curator":
		return 2
	default: // "user" or any other role
		return 1
	}
}

// autoPin evaluates the auto-pin rule for a field on an entity.
// In the assertions table, one assertion per (entity_id, field) is pinned.
//
// Priority order:
//  1. Recent curator > curator > recent user > user (by role priority DESC, id DESC)
//  2. No human assertion → highest-priority source (by source priority DESC)
//  3. No assertions → field absent
//
// Source priority is resolved by joining the source record table to get the
// source name, then looking up priority in the priorities map.
func autoPin(ctx context.Context, tx pgx.Tx, assertionsTable, entityIDCol string, entityID ID, field, sourceIDCol, sourceTable string, priorities map[string]int) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT a.id, a.user_id, a.role, a.pinned, st.source
		 FROM %s a
		 LEFT JOIN %s st ON a.%s = st.id
		 WHERE a.%s = $1 AND a.field = $2`,
		assertionsTable, sourceTable, sourceIDCol, entityIDCol,
	), entityID, field)
	if err != nil {
		return fmt.Errorf("autoPin: %w", err)
	}
	defer rows.Close()

	type row struct {
		id     int64
		human  bool
		role   string
		source string
		pinned bool
	}
	var assertions []row
	for rows.Next() {
		var r row
		var userID pgtype.UUID
		var role, source pgtype.Text
		if err := rows.Scan(&r.id, &userID, &role, &r.pinned, &source); err != nil {
			return fmt.Errorf("autoPin: %w", err)
		}
		r.human = userID.Valid
		if role.Valid {
			r.role = role.String
		}
		if source.Valid {
			r.source = source.String
		}
		assertions = append(assertions, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("autoPin: %w", err)
	}

	if len(assertions) == 0 {
		return nil
	}

	// Find the winner.
	// Human assertions: highest role priority, then most recent (highest id).
	// Source assertions: highest source priority.
	// Any human beats any source.
	winnerIdx := 0
	for i, a := range assertions[1:] {
		w := assertions[winnerIdx]
		idx := i + 1

		if a.human && !w.human {
			winnerIdx = idx
		} else if !a.human && w.human {
			// keep current winner
		} else if a.human && w.human {
			// Both human: compare role priority, then recency (id).
			ap, wp := rolePriority(a.role), rolePriority(w.role)
			if ap > wp || (ap == wp && a.id > w.id) {
				winnerIdx = idx
			}
		} else {
			// Both source: compare source priority.
			if priorities[a.source] > priorities[w.source] {
				winnerIdx = idx
			}
		}
	}

	winnerID := assertions[winnerIdx].id

	// Update pinned state.
	for _, a := range assertions {
		shouldPin := a.id == winnerID
		if a.pinned == shouldPin {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`UPDATE %s SET pinned = $1 WHERE id = $2`, assertionsTable,
		), shouldPin, a.id); err != nil {
			return fmt.Errorf("autoPin: %w", err)
		}
	}

	return nil
}
