package bbl

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// importUserSQL is the find-or-create CTE used by importUserBatch.
// $1=source, $2=source_record_id, $3=username, $4=email, $5=name, $6=role,
// $7=expires_at, $8=schemes[], $9=vals[], $10=authProvider, $11=newID.
const importUserSQL = `
	WITH
	updated AS (
		UPDATE bbl_users
		SET username = $3, email = $4, name = $5,
		    auth_providers = CASE
		        WHEN $10::text IS NOT NULL
		         AND NOT EXISTS (
		             SELECT 1 FROM jsonb_array_elements(auth_providers) p
		             WHERE p->>'provider' = $10::text
		         )
		        THEN auth_providers || jsonb_build_array(jsonb_build_object('provider', $10))
		        ELSE auth_providers
		    END
		WHERE id = (
		    SELECT user_id FROM bbl_user_sources
		    WHERE source = $1 AND source_record_id = $2
		)
		RETURNING id, created_at, username, email, name, role, deactivate_at, person_identity_id, auth_providers
	),
	created AS (
		INSERT INTO bbl_users (id, username, email, name, role, auth_providers)
		SELECT $11, $3, $4, $5, $6,
		       CASE WHEN $10::text IS NOT NULL
		            THEN jsonb_build_array(jsonb_build_object('provider', $10))
		            ELSE '[]'::jsonb
		       END
		WHERE NOT EXISTS (SELECT 1 FROM updated)
		RETURNING id, created_at, username, email, name, role, deactivate_at, person_identity_id, auth_providers
	),
	u AS (
		SELECT * FROM updated UNION ALL SELECT * FROM created
	),
	source_stamp AS (
		INSERT INTO bbl_user_sources (user_id, source, source_record_id, expires_at)
		SELECT id, $1, $2, $7 FROM u
		ON CONFLICT (user_id, source) DO UPDATE
			SET source_record_id = EXCLUDED.source_record_id,
			    last_seen_at     = transaction_timestamp(),
			    expires_at       = EXCLUDED.expires_at
	),
	del_idents AS (
		DELETE FROM bbl_user_identifiers
		WHERE user_id = (SELECT id FROM u)
		  AND source = $1
		  AND (scheme, val) NOT IN (
		      SELECT scheme, val FROM unnest($8::text[], $9::text[]) AS t(scheme, val)
		  )
	),
	new_idents AS (
		INSERT INTO bbl_user_identifiers (user_id, source, scheme, val)
		SELECT (SELECT id FROM u), $1, scheme, val
		FROM unnest($8::text[], $9::text[]) AS t(scheme, val)
		ON CONFLICT (user_id, source, scheme, val) DO NOTHING
	)
	SELECT * FROM u`

func importUserArgs(in *ImportUserInput) []any {
	schemes := make([]string, len(in.Identifiers))
	vals := make([]string, len(in.Identifiers))
	for i, ident := range in.Identifiers {
		schemes[i] = ident.Scheme
		vals[i] = ident.Val
	}
	var authProvider *string
	if in.AuthProvider != "" {
		authProvider = &in.AuthProvider
	}
	return []any{
		in.Source, in.SourceRecordID,
		in.Username, in.Email, in.Name, in.Role,
		in.ExpiresAt,
		schemes, vals,
		authProvider,
		newID(),
	}
}

// GetUser fetches a user by primary key. Returns ErrNotFound if no row exists.
func (r *Repo) GetUser(ctx context.Context, id string) (*User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, created_at, username, email, name, role, deactivate_at, person_identity_id, auth_providers
		FROM bbl_users
		WHERE id = $1`, id)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetUser: %w", err)
	}
	return u, nil
}

// GetUserByIdentifier looks up a user by auth claim (scheme, val).
// This is the primary login lookup path. Returns ErrNotFound if no match.
func (r *Repo) GetUserByIdentifier(ctx context.Context, scheme, val string) (*User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT u.id, u.created_at, u.username, u.email, u.name, u.role, u.deactivate_at, u.person_identity_id
		FROM bbl_users u
		JOIN bbl_user_identifiers i ON i.user_id = u.id
		WHERE i.scheme = $1 AND i.val = $2`, scheme, val)
	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetUserByIdentifier: %w", err)
	}
	return u, nil
}

// CreateUser inserts a new user. Intended for manual admin creation.
// Returns ErrConflict if the username is already taken.
func (r *Repo) CreateUser(ctx context.Context, attrs UserAttrs) (*User, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO bbl_users (id, username, email, name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, username, email, name, role, deactivate_at, person_identity_id, auth_providers`,
		newID(), attrs.Username, attrs.Email, attrs.Name, attrs.Role)
	u, err := scanUser(row)
	if isUniqueViolation(err) {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("CreateUser: %w", err)
	}
	return u, nil
}

// ImportUsers runs a full sweep from src, importing all records in batches.
// It returns the number of successfully imported users and the first fatal error
// encountered, whether from the source or the database.
// Use src.Iter directly for finer-grained error handling per entry.
func (r *Repo) ImportUsers(ctx context.Context, src UserSource) (int, error) {
	seq, err := src.Iter(ctx)
	if err != nil {
		return 0, fmt.Errorf("ImportUsers: %w", err)
	}

	const batchSize = 250
	var (
		pending []*ImportUserInput
		total   int
	)

	flush := func() error {
		n, err := r.importUserBatch(ctx, pending)
		total += n
		pending = pending[:0]
		return err
	}

	for in, err := range seq {
		if err != nil {
			return total, fmt.Errorf("ImportUsers: %w", err)
		}
		pending = append(pending, in)
		if len(pending) == batchSize {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if len(pending) > 0 {
		if err := flush(); err != nil {
			return total, err
		}
	}
	return total, nil
}

// importUserBatch sends a batch of import records in a single round trip.
// Concurrent writes within a batch sweep are assumed to be avoided by the scheduler,
// so there is no retry loop here.
func (r *Repo) importUserBatch(ctx context.Context, records []*ImportUserInput) (int, error) {
	b := &pgx.Batch{}
	for _, in := range records {
		b.Queue(importUserSQL, importUserArgs(in)...)
	}

	results := r.db.SendBatch(ctx, b)
	defer results.Close()

	var n int
	for _, in := range records {
		if _, err := scanUser(results.QueryRow()); err != nil {
			if isUniqueViolation(err) {
				return n, fmt.Errorf("ImportUsers: identifier conflict for source record %s/%s: %w",
					in.Source, in.SourceRecordID, ErrConflict)
			}
			return n, fmt.Errorf("ImportUsers: %w", err)
		}
		n++
	}
	return n, nil
}


func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(
		&u.ID,
		&u.CreatedAt,
		&u.Username,
		&u.Email,
		&u.Name,
		&u.Role,
		&u.DeactivateAt,
		&u.PersonIdentityID,
		&u.AuthProviders,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
