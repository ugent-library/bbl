package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// executeFieldWrites executes all field write operations via pgx.Batch.
// Two round-trips: one for assertions (history + delete + insert),
// one for extensions.
func executeFieldWrites(ctx context.Context, tx pgx.Tx, revID int64, user *User, muts []updater, effects []*updateEffect) error {
	// Collect non-noop field updaters.
	type fieldOp struct {
		mut updater
		eff *updateEffect
	}
	var ops []fieldOp
	for i, eff := range effects {
		if eff == nil {
			continue
		}
		switch muts[i].(type) {
		case *Set, *Hide, *Unset:
			ops = append(ops, fieldOp{muts[i], eff})
		}
	}
	if len(ops) == 0 {
		return nil
	}

	// --- Round-trip 1: history + delete + insert assertions ---
	batch := &pgx.Batch{}

	// 1. History: log old values.
	for _, op := range ops {
		rt, id, field := fieldOpTarget(op.mut)
		batch.Queue(fmt.Sprintf(
			`INSERT INTO bbl_history (rev_id, record_type, record_id, field, val, hidden)
			 SELECT $1, $2, a.%s, a.field, a.val, a.hidden
			 FROM %s a
			 WHERE a.%s = $3 AND a.field = $4 AND a.user_id IS NOT NULL`,
			entityIDCol(rt), assertionsTable(rt), entityIDCol(rt)),
			revID, rt, id, field)
	}

	// 2. Delete: remove old human assertions.
	for _, op := range ops {
		rt, id, field := fieldOpTarget(op.mut)
		batch.Queue(fmt.Sprintf(
			`DELETE FROM %s WHERE %s = $1 AND field = $2 AND user_id IS NOT NULL`,
			assertionsTable(rt), entityIDCol(rt)),
			id, field)
	}

	// 3. Insert: new assertion rows.
	// Track which inserts map to which extension-pending Set.
	type insertInfo struct {
		extIndex int // index into extPendings, or -1
	}
	type extPending struct {
		ft  *fieldType
		val any // original Go value for relation.buildInsert
		ids []int64
	}
	var inserts []insertInfo
	var extPendings []extPending

	for _, op := range ops {
		rt, id, field := fieldOpTarget(op.mut)
		switch m := op.mut.(type) {
		case *Set:
			ft, err := resolveFieldType(rt, field)
			if err != nil {
				return fmt.Errorf("executeFieldWrites: %w", err)
			}
			items, err := ft.marshal(m.Val)
			if err != nil {
				return fmt.Errorf("executeFieldWrites: marshal %s.%s: %w", rt, field, err)
			}
			extIdx := -1
			if ft.relation != nil {
				extIdx = len(extPendings)
				extPendings = append(extPendings, extPending{ft: ft, val: m.Val})
			}
			for _, item := range items {
				batch.Queue(fmt.Sprintf(
					`INSERT INTO %s (rev_id, %s, field, val, hidden, user_id, role)
					 VALUES ($1, $2, $3, $4, false, $5, $6) RETURNING id`,
					assertionsTable(rt), entityIDCol(rt)),
					revID, id, field, item, &user.ID, &user.Role)
				inserts = append(inserts, insertInfo{extIndex: extIdx})
			}
		case *Hide:
			batch.Queue(fmt.Sprintf(
				`INSERT INTO %s (rev_id, %s, field, val, hidden, user_id, role)
				 VALUES ($1, $2, $3, NULL, true, $4, $5) RETURNING id`,
				assertionsTable(rt), entityIDCol(rt)),
				revID, id, field, &user.ID, &user.Role)
			inserts = append(inserts, insertInfo{extIndex: -1})
		// Unset: no insert
		}
	}

	results := tx.SendBatch(ctx, batch)

	// Consume history results.
	for range ops {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("executeFieldWrites: history: %w", err)
		}
	}
	// Consume delete results.
	for range ops {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("executeFieldWrites: delete: %w", err)
		}
	}

	// Read insert RETURNING ids, collect assertion IDs per extension pending.
	for _, ins := range inserts {
		var aID int64
		if err := results.QueryRow().Scan(&aID); err != nil {
			results.Close()
			return fmt.Errorf("executeFieldWrites: insert: %w", err)
		}
		if ins.extIndex >= 0 {
			extPendings[ins.extIndex].ids = append(extPendings[ins.extIndex].ids, aID)
		}
	}

	if err := results.Close(); err != nil {
		return fmt.Errorf("executeFieldWrites: close: %w", err)
	}

	// --- Round-trip 2: extension inserts ---
	if len(extPendings) > 0 {
		extBatch := &pgx.Batch{}
		for _, ep := range extPendings {
			sql, args := ep.ft.relation.buildInsert(ep.ids, ep.val)
			if sql != "" {
				extBatch.Queue(sql, args...)
			}
		}
		if extBatch.Len() > 0 {
			extResults := tx.SendBatch(ctx, extBatch)
			for i := 0; i < extBatch.Len(); i++ {
				if _, err := extResults.Exec(); err != nil {
					extResults.Close()
					return fmt.Errorf("executeFieldWrites: extension: %w", err)
				}
			}
			if err := extResults.Close(); err != nil {
				return fmt.Errorf("executeFieldWrites: close ext: %w", err)
			}
		}
	}

	return nil
}

// fieldOpTarget extracts recordType, recordID, field from a field updater.
func fieldOpTarget(m updater) (string, ID, string) {
	switch u := m.(type) {
	case *Set:
		return u.RecordType, u.RecordID, u.Field
	case *Hide:
		return u.RecordType, u.RecordID, u.Field
	case *Unset:
		return u.RecordType, u.RecordID, u.Field
	}
	return "", ID{}, ""
}
