package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// autoPin evaluates the auto-pin rule for a field on an entity.
// In the assertions table, one assertion per (entity_id, field) is pinned.
//
// Rule:
//  1. Human assertion exists → it is pinned.
//  2. No human assertion → highest-priority source assertion is pinned.
//
// Source priority is resolved by joining the source record table to get the
// source name, then looking up priority in the priorities map.
func autoPin(ctx context.Context, tx pgx.Tx, assertionsTable, entityIDCol string, entityID ID, field, sourceIDCol, sourceTable string, priorities map[string]int) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT a.id, a.user_id, a.pinned, st.source
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
		id     ID
		human  bool
		source string
		pinned bool
	}
	var assertions []row
	for rows.Next() {
		var r row
		var userID pgtype.UUID
		var source pgtype.Text
		if err := rows.Scan(&r.id, &userID, &r.pinned, &source); err != nil {
			return fmt.Errorf("autoPin: %w", err)
		}
		r.human = userID.Valid
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

	// Rule 1: human assertion wins.
	humanIdx := -1
	for i, a := range assertions {
		if a.human {
			humanIdx = i
			break
		}
	}

	var winnerID ID
	if humanIdx >= 0 {
		winnerID = assertions[humanIdx].id
	} else {
		// Rule 2: highest-priority source wins.
		winnerIdx := 0
		winnerPri := priorities[assertions[0].source]
		for i, a := range assertions[1:] {
			p := priorities[a.source]
			if p > winnerPri {
				winnerIdx = i + 1
				winnerPri = p
			}
		}
		winnerID = assertions[winnerIdx].id
	}

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
