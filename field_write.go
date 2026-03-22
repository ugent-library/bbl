package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// executeFieldWrites executes all field write operations for human edits.
// Adds history + delete items to a batch, then delegates to writeAssertionRows
// for the insert + extension pipeline (shared with imports).
// Two round-trips: one for history + delete + inserts, one for extensions.
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

	batch := &pgx.Batch{}

	// History: log old values before deletion.
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

	// Delete: remove old human assertions.
	for _, op := range ops {
		rt, id, field := fieldOpTarget(op.mut)
		batch.Queue(fmt.Sprintf(
			`DELETE FROM %s WHERE %s = $1 AND field = $2 AND user_id IS NOT NULL`,
			assertionsTable(rt), entityIDCol(rt)),
			id, field)
	}

	preCount := batch.Len()

	// Build assertion rows for Set/Hide (Unset = delete only, no insert).
	var rows []assertionRow
	for _, op := range ops {
		rt, id, field := fieldOpTarget(op.mut)
		switch m := op.mut.(type) {
		case *Set:
			rows = append(rows, assertionRow{
				recordType: rt,
				recordID:   id,
				field:      field,
				val:        m.Val,
				userID:     &user.ID,
				role:       &user.Role,
			})
		case *Hide:
			rows = append(rows, assertionRow{
				recordType: rt,
				recordID:   id,
				field:      field,
				hidden:     true,
				userID:     &user.ID,
				role:       &user.Role,
			})
		}
	}

	return writeAssertionRows(ctx, tx, batch, preCount, revID, rows)
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
