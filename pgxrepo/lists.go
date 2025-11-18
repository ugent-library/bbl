package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/fracdex"
)

func (r *Repo) GetUserLists(ctx context.Context, userID string) ([]*bbl.List, error) {
	return getRows(ctx, r.conn, `
		SELECT `+listCols+`
		FROM bbl_lists_view l
		WHERE created_by_id = $1;`,
		[]any{userID},
		scanList)
}

func (r *Repo) GetList(ctx context.Context, id string) (*bbl.List, error) {
	return getRow(ctx, r.conn, `
		SELECT `+listCols+`
		FROM bbl_lists_view l
		WHERE id = $1;`,
		[]any{id},
		scanList)
}

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

func (r *Repo) DeleteList(ctx context.Context, id string) error {
	q := `DELETE FROM bbl_lists WHERE id = $1;`

	if _, err := r.conn.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("DeleteList: %w", err)
	}

	return nil
}

const listCols = `
	l.id,
	l.name,
	l.public,
	l.created_at,
	l.created_by_id,
	l.created_by
`

func scanList(row pgx.Row) (*bbl.List, error) {
	var rec bbl.List

	var createdByID *string
	var rawCreatedBy json.RawMessage

	if err := row.Scan(
		&rec.ID,
		&rec.Name,
		&rec.Public,
		&rec.CreatedAt,
		&createdByID,
		&rawCreatedBy,
	); err != nil {
		return nil, err
	}

	if createdByID != nil {
		rec.CreatedByID = *createdByID
	}
	if rawCreatedBy != nil {
		if err := json.Unmarshal(rawCreatedBy, &rec.CreatedBy); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}

func (r *Repo) GetListItems(ctx context.Context, listID string) ([]*bbl.ListItem, error) {
	return getRows(ctx, r.conn,
		`SELECT `+listItemCols+` FROM bbl_list_items_view l_i WHERE list_id = $1 ORDER BY pos ASC LIMIT 50;`,
		[]any{listID},
		scanListItem)
}

func (r *Repo) ListItemsIter(ctx context.Context, listID string, errPtr *error) iter.Seq[*bbl.ListItem] {
	return rowsIter(ctx, r.conn, errPtr,
		`SELECT `+listItemCols+` FROM bbl_list_items_view l_i WHERE list_id = $1 ORDER BY pos ASC;`,
		[]any{listID},
		scanListItem)
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

const listItemCols = `
	l_i.pos,
	l_i.work_id,
	l_i.work
`

func scanListItem(row pgx.Row) (*bbl.ListItem, error) {
	var rec bbl.ListItem

	var rawWork json.RawMessage

	if err := row.Scan(
		&rec.Pos,
		&rec.WorkID,
		&rawWork,
	); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawWork, &rec.Work); err != nil {
		return nil, err
	}

	return &rec, nil
}
