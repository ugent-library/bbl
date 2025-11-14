package pgxrepo

import (
	"context"
	"fmt"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/fracdex"
)

func (r *Repo) CreateList(ctx context.Context, userID, name string) (string, error) {
	q := `
		INSERT INTO bbl_lists (id, name, created_by_id)
		VALUES ($1, $2, nullif($3, '')::uuid);`

	id := bbl.NewID()

	if _, err := r.conn.Exec(ctx, q, id, name, userID); err != nil {
		return "", fmt.Errorf("CreateList: %w", err)
	}

	return id, nil
}

func (r *Repo) GetUserLists(ctx context.Context, userID string) ([]*bbl.List, error) {
	q := `
		SELECT id, name, public, created_at, created_by_id
		FROM bbl_lists
		WHERE created_by_id = $1;`

	rows, err := r.conn.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUserLists: %w", err)
	}

	recs, err := pgx.CollectRows(rows, scanList)
	if err != nil {
		return nil, fmt.Errorf("GetUserLists: %w", err)
	}

	return recs, nil
}

func (r *Repo) ListItemsIter(ctx context.Context, listID string, errPtr *error) iter.Seq[*bbl.ListItem] {
	q := `SELECT work_id, pos FROM bbl_list_items WHERE list_id = $1 ORDER BY pos ASC;`

	return func(yield func(*bbl.ListItem) bool) {
		rows, err := r.conn.Query(ctx, q, listID)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanListItem(rows)
			if err != nil {
				*errPtr = err
				return
			}
			if !yield(rec) {
				return
			}
		}
	}
}

func (r *Repo) AddListItems(ctx context.Context, listID string, workIDs []string) error {
	var pos string

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("AddListItems: %w", err)
	}

	var posPtr *string

	if err := tx.QueryRow(ctx, `SELECT max(pos) FROM bbl_list_items WHERE list_id = $1`, listID).Scan(&posPtr); err != nil {
		return fmt.Errorf("AddListItems: %w", err)
	}

	if posPtr != nil {
		pos = *posPtr
	}

	positions, err := fracdex.NKeysBetween(pos, "", uint(len(workIDs)))
	if err != nil {
		return fmt.Errorf("AddListItems: %w", err)
	}

	rows := make([][]any, len(workIDs))

	for i, workID := range workIDs {
		rows[i] = []any{listID, workID, positions[i]}
	}

	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"bbl_list_items"},
		[]string{"list_id", "work_id", "pos"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("AddListItems: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("AddListItems: %w", err)
	}

	return nil
}

func scanList(row pgx.CollectableRow) (*bbl.List, error) {
	var rec bbl.List

	var createdByID *string

	if err := row.Scan(
		&rec.ID,
		&rec.Name,
		&rec.Public,
		&rec.CreatedAt,
		&createdByID,
	); err != nil {
		return nil, err
	}

	if createdByID != nil {
		rec.CreatedByID = *createdByID
	}

	return &rec, nil
}

func scanListItem(row pgx.CollectableRow) (*bbl.ListItem, error) {
	var rec bbl.ListItem

	if err := row.Scan(
		&rec.WorkID,
		&rec.Pos,
	); err != nil {
		return nil, err
	}

	return &rec, nil
}
