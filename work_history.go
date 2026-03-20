package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkHistoryEntry is one entry in the history view for a work.
// Combines current assertions + history table entries.
type WorkHistoryEntry struct {
	RevID     int64
	RevAt     time.Time
	Field     string
	Val       json.RawMessage
	Hidden    bool
	UserID    *ID
	Role      string
	Source    string
	Pinned    bool
	IsHistory bool // true = from bbl_history (old value), false = current assertion
}

// GetWorkHistory returns the full history for a work: current assertions
// plus historical values from bbl_history, ordered by field then rev_id desc.
func (r *Repo) GetWorkHistory(ctx context.Context, workID ID) ([]WorkHistoryEntry, error) {
	rows, err := r.db.Query(ctx, `
		SELECT sub.rev_id, r.created_at, sub.field, sub.val, sub.hidden,
		       sub.user_id, sub.role, sub.source, sub.pinned, sub.is_history
		FROM (
			-- Current assertions
			SELECT a.rev_id, a.field, a.val, a.hidden,
			       a.user_id, a.role, s.source, a.pinned,
			       false AS is_history
			FROM bbl_work_assertions a
			LEFT JOIN bbl_work_sources s ON s.id = a.work_source_id
			WHERE a.work_id = $1

			UNION ALL

			-- Historical values (replaced by human edits)
			SELECT h.rev_id, h.field, h.val, h.hidden,
			       NULL::uuid AS user_id, NULL AS role, NULL AS source,
			       false AS pinned,
			       true AS is_history
			FROM bbl_history h
			WHERE h.record_type = 'work' AND h.record_id = $1
		) sub
		JOIN bbl_revs r ON r.id = sub.rev_id
		ORDER BY sub.field, sub.rev_id DESC`,
		workID)
	if err != nil {
		return nil, fmt.Errorf("GetWorkHistory: %w", err)
	}
	defer rows.Close()

	var result []WorkHistoryEntry
	for rows.Next() {
		var e WorkHistoryEntry
		var userID pgtype.UUID
		var role, source pgtype.Text
		if err := rows.Scan(
			&e.RevID, &e.RevAt, &e.Field, &e.Val, &e.Hidden,
			&userID, &role, &source, &e.Pinned,
			&e.IsHistory,
		); err != nil {
			return nil, fmt.Errorf("GetWorkHistory: %w", err)
		}
		if userID.Valid {
			id := ID(userID.Bytes)
			e.UserID = &id
		}
		if role.Valid {
			e.Role = role.String
		}
		if source.Valid {
			e.Source = source.String
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
