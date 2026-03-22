package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// assertionRow is a single field assertion to be inserted.
// Both import and update paths produce these, then feed them
// to writeAssertionRows for the shared marshal → insert → extension pipeline.
type assertionRow struct {
	recordType     string
	recordID       ID
	field          string
	val            any     // Go value matching fieldType; nil when hidden
	hidden         bool
	sourceRecordID *ID // set for source imports
	userID         *ID // set for human edits
	role           *string
}

// assertionRowFields reconstructs the field→value map from assertion rows.
// Used to feed validateRecord before writing.
func assertionRowFields(rows []assertionRow) map[string]any {
	fields := make(map[string]any)
	for _, r := range rows {
		if !r.hidden {
			fields[r.field] = r.val
		}
	}
	return fields
}

// writeAssertionRows marshals values, inserts assertion rows, and writes
// extension rows for FK-bearing types.
//
// Pipeline:
//  1. Resolve fieldType, marshal each value → []json.RawMessage
//  2. Queue INSERT ... RETURNING id into batch (after any pre-items the caller added)
//  3. Send batch, consume preCount results (caller's history/deletes), collect assertion IDs
//  4. Build and send extension batch for FK-bearing types
//
// preCount is the number of items already in batch whose results the caller
// needs consumed (e.g. history + delete in the update path). Pass 0 for imports.
func writeAssertionRows(ctx context.Context, tx pgx.Tx, batch *pgx.Batch, preCount int, revID int64, rows []assertionRow) error {
	// --- Phase 1: marshal and queue assertion INSERTs ---
	type insertMeta struct {
		groupIdx int // index into groups; -1 if no extension
	}
	type group struct {
		ft  *fieldType
		val any
		ids []int64
	}
	var inserts []insertMeta
	var groups []group

	for _, r := range rows {
		ft, err := resolveFieldType(r.recordType, r.field)
		if err != nil {
			return fmt.Errorf("writeAssertionRows: %w", err)
		}

		table := assertionsTable(r.recordType)
		entityCol := entityIDCol(r.recordType)
		srcCol := sourceIDCol(r.recordType)

		if r.hidden {
			batch.Queue(fmt.Sprintf(
				`INSERT INTO %s (rev_id, %s, field, val, hidden, %s, user_id, role)
				 VALUES ($1, $2, $3, NULL, true, $4, $5, $6) RETURNING id`,
				table, entityCol, srcCol),
				revID, r.recordID, r.field, r.sourceRecordID, r.userID, r.role)
			inserts = append(inserts, insertMeta{groupIdx: -1})
			continue
		}

		items, err := ft.marshal(r.val)
		if err != nil {
			return fmt.Errorf("writeAssertionRows: marshal %s.%s: %w", r.recordType, r.field, err)
		}

		gIdx := -1
		if ft.relation != nil {
			gIdx = len(groups)
			groups = append(groups, group{ft: ft, val: r.val})
		}

		for _, item := range items {
			batch.Queue(fmt.Sprintf(
				`INSERT INTO %s (rev_id, %s, field, val, hidden, %s, user_id, role)
				 VALUES ($1, $2, $3, $4, false, $5, $6, $7) RETURNING id`,
				table, entityCol, srcCol),
				revID, r.recordID, r.field, item, r.sourceRecordID, r.userID, r.role)
			inserts = append(inserts, insertMeta{groupIdx: gIdx})
		}
	}

	if preCount == 0 && len(inserts) == 0 {
		return nil
	}

	// --- Phase 2: send batch, consume pre-items, collect assertion IDs ---
	results := tx.SendBatch(ctx, batch)

	for i := 0; i < preCount; i++ {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("writeAssertionRows: pre %d: %w", i, err)
		}
	}

	for _, ins := range inserts {
		var aID int64
		if err := results.QueryRow().Scan(&aID); err != nil {
			results.Close()
			return fmt.Errorf("writeAssertionRows: insert: %w", err)
		}
		if ins.groupIdx >= 0 {
			groups[ins.groupIdx].ids = append(groups[ins.groupIdx].ids, aID)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("writeAssertionRows: close: %w", err)
	}

	// --- Phase 3: extension inserts for FK-bearing types ---
	extBatch := &pgx.Batch{}
	for _, g := range groups {
		sql, args := g.ft.relation.buildInsert(g.ids, g.val)
		if sql != "" {
			extBatch.Queue(sql, args...)
		}
	}
	if extBatch.Len() > 0 {
		extResults := tx.SendBatch(ctx, extBatch)
		for i := 0; i < extBatch.Len(); i++ {
			if _, err := extResults.Exec(); err != nil {
				extResults.Close()
				return fmt.Errorf("writeAssertionRows: extension: %w", err)
			}
		}
		if err := extResults.Close(); err != nil {
			return fmt.Errorf("writeAssertionRows: close ext: %w", err)
		}
	}

	return nil
}
