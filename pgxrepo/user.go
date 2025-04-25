package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) GetUser(ctx context.Context, id string) (*bbl.User, error) {
	return getUser(ctx, r.conn, id)
}

func (r *Repo) UsersIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.User] {
	q := `
		select id, username, email, name, created_at, updated_at, identifiers
		from bbl_users_view;`

	return func(yield func(*bbl.User) bool) {
		rows, err := r.conn.Query(ctx, q)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanUser(rows)
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

func getUserByColQuery(col string) string {
	return `select id, username, email, name, created_at, updated_at, identifiers
		    from bbl_users_view
		    where ` + col + ` = $1;`
}

func getUser(ctx context.Context, conn pgxConn, id string) (*bbl.User, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		switch scheme {
		case "username":
			row = conn.QueryRow(ctx, getUserByColQuery("username"), val)
		case "email":
			row = conn.QueryRow(ctx, getUserByColQuery("email"), val)
		default:
			row = conn.QueryRow(ctx, `
				select u.id, u.username, u.email, u.name, u.created_at, u.updated_at, u.identifiers
				from bbl_users_view u, bbl_user_identifiers u_i
				where u.id = u_i.user_id and u_i.scheme = $1 and u_i.val = $2;`,
				scheme, val,
			)
		}
	} else {
		row = conn.QueryRow(ctx, getUserByColQuery("id"), id)
	}

	rec, err := scanUser(row)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetUser %s: %w", id, err)
	}

	return rec, err
}

func scanUser(row pgx.Row) (*bbl.User, error) {
	var rec bbl.User
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Username, &rec.Email, &rec.Name, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
