package pgxrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) GetUser(ctx context.Context, id string) (*bbl.User, error) {
	return getUser(ctx, r.conn, id)
}

// TODO use func() error instead of error pointer
func (r *Repo) UsersIter(ctx context.Context, errPtr *error) iter.Seq[*bbl.User] {
	q := `select ` + userCols + ` from bbl_users_view u;`

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

// TODO handle NULL deactivate_at
func (r *Repo) SaveUser(ctx context.Context, rec *bbl.User) error {
	if err := rec.Validate(); err != nil {
		return fmt.Errorf("SaveUser: %w", err)
	}

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("SaveUser: %s", err)
	}
	defer tx.Rollback(ctx)

	var update bool

	if rec.ID == "" {
		rec.ID = bbl.NewID()
	} else {
		oldRec, err := getUser(ctx, tx, rec.ID)
		if err != nil && !errors.Is(err, bbl.ErrNotFound) {
			return fmt.Errorf("SaveUser: %s", err)
		}
		if oldRec != nil {
			update = true
			rec.ID = oldRec.ID
		} else {
			rec.ID = bbl.NewID()
		}
	}

	if update {
		_, err = tx.Exec(ctx, `
			update bbl_users
			set username = $2,
			    email = $3,
			    name = $4,
			    updated_at = transaction_timestamp(),
			    deactivate_at = $5
			where id = $1;`,
			rec.ID, rec.Username, rec.Email, rec.Name, rec.DeactivateAt,
		)
		if err != nil {
			return fmt.Errorf("SaveUser: %s (%+v)", err, rec)
		}
		_, err = tx.Exec(ctx, `
			delete from bbl_user_identifiers where user_id = $1`,
			rec.ID,
		)
		if err != nil {
			return fmt.Errorf("SaveUser: %s (%+v)", err, rec)
		}
	} else {
		_, err = tx.Exec(ctx, `
			insert into bbl_users (id, username, email, name, role, deactivate_at)
			values ($1, $2, $3, $4, $5, $6)`,
			rec.ID, rec.Username, rec.Email, rec.Name, bbl.UserRole, rec.DeactivateAt,
		)
		if err != nil {
			return fmt.Errorf("SaveUser: %s (%+v)", err, rec)
		}
	}

	for i, ident := range rec.Identifiers {
		_, err = tx.Exec(ctx, `
			insert into bbl_user_identifiers (user_id, idx, scheme, val)
			values ($1, $2, $3, $4)`,
			rec.ID, i, ident.Scheme, ident.Val,
		)
		if err != nil {
			return fmt.Errorf("SaveUser: %s (%+v)", err, rec)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("SaveUser: %w", err)
	}

	return nil
}

func getUser(ctx context.Context, conn pgxConn, id string) (*bbl.User, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		switch scheme {
		case "username":
			row = conn.QueryRow(ctx, `select `+userCols+` from bbl_users_view u where u.username = $1;`, val)
		case "email":
			row = conn.QueryRow(ctx, `select `+userCols+` from bbl_users_view u where u.email = $1;`, val)
		default:
			row = conn.QueryRow(ctx, `
				select `+userCols+`
				from bbl_users_view u, bbl_user_identifiers u_i
				where u.id = u_i.user_id and u_i.scheme = $1 and u_i.val = $2;`,
				scheme, val,
			)
		}
	} else {
		row = conn.QueryRow(ctx, `select `+userCols+` from bbl_users_view u where u.id = $1;`, id)
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

const userCols = `
	u.id,
	u.username,
	u.email,
	u.name,
	u.role,
	u.created_at,
	u.updated_at,
	u.deactivate_at,
	u.identifiers
`

func scanUser(row pgx.Row) (*bbl.User, error) {
	var rec bbl.User
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Username, &rec.Email, &rec.Name, &rec.Role, &rec.CreatedAt, &rec.UpdatedAt, &rec.DeactivateAt, &rawIdentifiers); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}
