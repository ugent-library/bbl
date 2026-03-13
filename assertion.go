package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// autoPinScalar evaluates auto-pin for a scalar field grouping key.
// Within (entityID, field), the rule is:
//  1. If a human assertion exists (user_id IS NOT NULL) → it is pinned, done.
//  2. Otherwise → highest-priority source's assertion wins.
//
// Source priority is resolved by joining through the entity's sources table
// (e.g. bbl_work_sources) to get the source name, then looking up priority.
//
// Parameters:
//   - table: assertion table (e.g. "bbl_work_fields")
//   - entityIDCol: entity FK column (e.g. "work_id")
//   - sourceIDCol: source FK column (e.g. "work_source_id")
//   - sourceTable: entity sources table (e.g. "bbl_work_sources")
func autoPinScalar(ctx context.Context, tx pgx.Tx, table, entityIDCol string, entityID ID, field, sourceIDCol, sourceTable string, priorities map[string]int) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT a.id, a.user_id, a.pinned, st.source
		 FROM %s a
		 LEFT JOIN %s st ON a.%s = st.id
		 WHERE a.%s = $1 AND a.field = $2`,
		table, sourceTable, sourceIDCol, entityIDCol,
	), entityID, field)
	if err != nil {
		return fmt.Errorf("autoPinScalar: %w", err)
	}
	defer rows.Close()

	type row struct {
		id     ID
		human  bool   // user_id IS NOT NULL
		source string // source name (empty for human assertions)
		pinned bool
	}
	var assertions []row
	for rows.Next() {
		var r row
		var userID pgtype.UUID
		var source pgtype.Text
		if err := rows.Scan(&r.id, &userID, &r.pinned, &source); err != nil {
			return fmt.Errorf("autoPinScalar: %w", err)
		}
		r.human = userID.Valid
		if source.Valid {
			r.source = source.String
		}
		assertions = append(assertions, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("autoPinScalar: %w", err)
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
			`UPDATE %s SET pinned = $1 WHERE id = $2`, table,
		), shouldPin, a.id); err != nil {
			return fmt.Errorf("autoPinScalar: %w", err)
		}
	}

	return nil
}

// autoPinCollective evaluates auto-pin for a collective grouping key.
// Within (entityID) per table, the rule is:
//  1. If any human assertions exist → all human rows pinned, all others unpinned.
//  2. Otherwise → all rows from highest-priority source pinned, rest unpinned.
//
// Used for identifiers, classifications, contributors, titles, abstracts,
// lay summaries, notes, keywords, and FK relations.
func autoPinCollective(ctx context.Context, tx pgx.Tx, table, entityIDCol string, entityID ID, sourceIDCol, sourceTable string, priorities map[string]int) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT a.id, a.user_id, a.pinned, st.source
		 FROM %s a
		 LEFT JOIN %s st ON a.%s = st.id
		 WHERE a.%s = $1`,
		table, sourceTable, sourceIDCol, entityIDCol,
	), entityID)
	if err != nil {
		return fmt.Errorf("autoPinCollective: %w", err)
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
			return fmt.Errorf("autoPinCollective: %w", err)
		}
		r.human = userID.Valid
		if source.Valid {
			r.source = source.String
		}
		assertions = append(assertions, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("autoPinCollective: %w", err)
	}

	if len(assertions) == 0 {
		return nil
	}

	// Rule 1: if any human assertions exist, pin all human rows.
	hasHuman := false
	for _, a := range assertions {
		if a.human {
			hasHuman = true
			break
		}
	}

	if hasHuman {
		for _, a := range assertions {
			shouldPin := a.human
			if a.pinned == shouldPin {
				continue
			}
			if _, err := tx.Exec(ctx, fmt.Sprintf(
				`UPDATE %s SET pinned = $1 WHERE id = $2`, table,
			), shouldPin, a.id); err != nil {
				return fmt.Errorf("autoPinCollective: %w", err)
			}
		}
		return nil
	}

	// Rule 2: highest-priority source wins. Pin all rows from winner.
	winnerSource := assertions[0].source
	winnerPri := priorities[assertions[0].source]
	for _, a := range assertions[1:] {
		p := priorities[a.source]
		if p > winnerPri {
			winnerSource = a.source
			winnerPri = p
		}
	}

	for _, a := range assertions {
		shouldPin := a.source == winnerSource
		if a.pinned == shouldPin {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`UPDATE %s SET pinned = $1 WHERE id = $2`, table,
		), shouldPin, a.id); err != nil {
			return fmt.Errorf("autoPinCollective: %w", err)
		}
	}

	return nil
}
