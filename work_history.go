package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// WorkAssertion is one assertion row for the history view.
type WorkAssertion struct {
	ID         int64
	RevID      int64
	Field      string
	Val        json.RawMessage // nil for collectives and hides
	Hidden     bool
	UserID     *ID
	Role       string
	Source     string // empty for human assertions
	AssertedAt time.Time
	Pinned     bool
	Size       int    // for collectives: number of relation rows
}

// GetWorkHistory returns all assertions for a work, ordered by field
// then recency (newest first). Collective assertions include a summary
// count from their relation tables.
func (r *Repo) GetWorkHistory(ctx context.Context, workID ID) ([]WorkAssertion, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.rev_id, a.field, a.val, a.hidden,
		       a.user_id, a.role, a.asserted_at, a.pinned,
		       s.source,
		       COALESCE(t.n, 0) + COALESCE(ab.n, 0) + COALESCE(ls.n, 0) +
		       COALESCE(n.n, 0) + COALESCE(kw.n, 0) + COALESCE(i.n, 0) +
		       COALESCE(cl.n, 0) + COALESCE(co.n, 0) + COALESCE(p.n, 0) +
		       COALESCE(o.n, 0) + COALESCE(rl.n, 0) AS rel_count
		FROM bbl_work_assertions a
		LEFT JOIN bbl_work_sources s ON s.id = a.work_source_id
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_titles WHERE assertion_id = a.id) t ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_abstracts WHERE assertion_id = a.id) ab ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_lay_summaries WHERE assertion_id = a.id) ls ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_notes WHERE assertion_id = a.id) n ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_keywords WHERE assertion_id = a.id) kw ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_identifiers WHERE assertion_id = a.id) i ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_classifications WHERE assertion_id = a.id) cl ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_contributors WHERE assertion_id = a.id) co ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_projects WHERE assertion_id = a.id) p ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_organizations WHERE assertion_id = a.id) o ON true
		LEFT JOIN LATERAL (SELECT count(*) AS n FROM bbl_work_rels WHERE assertion_id = a.id) rl ON true
		WHERE a.work_id = $1
		ORDER BY a.field, a.id DESC`,
		workID)
	if err != nil {
		return nil, fmt.Errorf("GetWorkHistory: %w", err)
	}
	defer rows.Close()

	var result []WorkAssertion
	for rows.Next() {
		var a WorkAssertion
		var userID pgtype.UUID
		var role, source pgtype.Text
		var relCount int
		if err := rows.Scan(
			&a.ID, &a.RevID, &a.Field, &a.Val, &a.Hidden,
			&userID, &role, &a.AssertedAt, &a.Pinned,
			&source, &relCount,
		); err != nil {
			return nil, fmt.Errorf("GetWorkHistory: %w", err)
		}
		if userID.Valid {
			id := ID(userID.Bytes)
			a.UserID = &id
		}
		if role.Valid {
			a.Role = role.String
		}
		if source.Valid {
			a.Source = source.String
		}
		if a.Val != nil {
			a.Size = 1
		} else {
			a.Size = relCount
		}
		result = append(result, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetWorkHistory: %w", err)
	}
	return result, nil
}
