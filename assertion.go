package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type autoPinRow struct {
	id             int64
	human          bool
	sourceRecordID *ID
	source         string
	pinned         bool
}

// autoPin evaluates the auto-pin rule for a field on an entity.
//
// All assertions for the winning asserter are pinned (exclusive mode).
// For scalars (one row per asserter), this pins exactly one row.
// For collections (multiple rows per asserter), all items from the winner are pinned.
//
// Priority: human > source by priority. One human assertion per field
// (replace semantics), so no role comparison needed among humans.
func autoPin(ctx context.Context, tx pgx.Tx, assertionsTable, entityIDCol string, entityID ID, field, sourceIDCol, sourceTable string, priorities map[string]int) error {
	rows, err := tx.Query(ctx, fmt.Sprintf(
		`SELECT a.id, a.user_id, a.%s, a.pinned, st.source
		 FROM %s a
		 LEFT JOIN %s st ON a.%s = st.id
		 WHERE a.%s = $1 AND a.field = $2`,
		sourceIDCol, assertionsTable, sourceTable, sourceIDCol, entityIDCol,
	), entityID, field)
	if err != nil {
		return fmt.Errorf("autoPin: %w", err)
	}
	defer rows.Close()

	var assertions []autoPinRow
	for rows.Next() {
		var r autoPinRow
		var userID, srcRecID pgtype.UUID
		var source pgtype.Text
		if err := rows.Scan(&r.id, &userID, &srcRecID, &r.pinned, &source); err != nil {
			return fmt.Errorf("autoPin: %w", err)
		}
		r.human = userID.Valid
		if srcRecID.Valid {
			id := ID(srcRecID.Bytes)
			r.sourceRecordID = &id
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

	// Pin all rows from the winning asserter.
	// For scalars (one row per asserter), this pins exactly one row.
	// For collections (multiple rows per asserter), all items from the winner are pinned.
	return autoPinExclusive(ctx, tx, assertionsTable, assertions, priorities)
}

// autoPinExclusive pins all items from the winning asserter.
// Winner: human (if exists), else highest-priority source.
func autoPinExclusive(ctx context.Context, tx pgx.Tx, assertionsTable string, assertions []autoPinRow, priorities map[string]int) error {
	// Find the winning asserter.
	// For humans: user_id IS NOT NULL (only one human per field).
	// For sources: pick the source with highest priority.
	hasHuman := false
	var winnerSourceRecordID *ID
	bestSourcePriority := -1

	for _, a := range assertions {
		if a.human {
			hasHuman = true
			break
		}
		if a.sourceRecordID != nil {
			sp := priorities[a.source]
			if sp > bestSourcePriority {
				bestSourcePriority = sp
				winnerSourceRecordID = a.sourceRecordID
			}
		}
	}

	for _, a := range assertions {
		var shouldPin bool
		if hasHuman {
			shouldPin = a.human
		} else {
			shouldPin = a.sourceRecordID != nil && winnerSourceRecordID != nil && *a.sourceRecordID == *winnerSourceRecordID
		}
		if a.pinned == shouldPin {
			continue
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`UPDATE %s SET pinned = $1 WHERE id = $2`, assertionsTable,
		), shouldPin, a.id); err != nil {
			return fmt.Errorf("autoPinExclusive: %w", err)
		}
	}
	return nil
}
