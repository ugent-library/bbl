package pgxrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/catbird"
)

func (r *Repo) AddRev(ctx context.Context, rev *bbl.Rev) error {
	revID := bbl.NewID()

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("AddRev: %s", err)
	}
	defer tx.Rollback(ctx)

	batch := &pgx.Batch{}

	batch.Queue(`
		INSERT INTO bbl_revs (id, user_id)
		VALUES ($1, nullif($2, '')::uuid);`,
		revID, rev.UserID,
	)

	for _, action := range rev.Actions {
		switch a := action.(type) {
		// TODO handle NULL deactivate_at
		// TODO this creates a lot of deactivate_at changes
		case *bbl.SaveUser:
			rec := a.User

			// validate
			if err := rec.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			var oldRec *bbl.User

			if rec.ID != "" {
				oldRec, err = getUser(ctx, tx, rec.ID)
				if err != nil && !errors.Is(err, bbl.ErrNotFound) {
					return fmt.Errorf("AddRev: %s", err)
				}
			}

			// conflict detection
			if oldRec != nil && a.MatchVersion && rec.Version != oldRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, rec.Version, oldRec.Version)
			}

			if oldRec != nil {
				rec.ID = oldRec.ID
			} else {
				rec.ID = bbl.NewID()
			}

			if oldRec == nil {
				batch.Queue(`
					INSERT INTO bbl_users (id, username, email, name, role, deactivate_at, version, created_by_id, updated_by_id)
					VALUES ($1, $2, $3, $4, $5, $6, 1, nullif($7, '')::uuid, nullif($8, '')::uuid);`,
					rec.ID, rec.Username, rec.Email, rec.Name, rec.Role, rec.DeactivateAt, rev.UserID, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "user", rec.ID, nil, rec.Identifiers)

				enqueueInsertChange(batch, "user", revID, rec.ID, rec.Diff(&bbl.User{}))
			} else {
				// only update if there are changes
				diff := rec.Diff(oldRec)
				if len(diff) == 0 {
					continue
				}

				batch.Queue(`
					UPDATE bbl_users
					SET username = $2,
						email = $3,
						name = $4,
						deactivate_at = $5,
						version = version + 1,
						updated_at = transaction_timestamp(),
						updated_by_id = nullif($6, '')::uuid
					WHERE id = $1;`,
					rec.ID, rec.Username, rec.Email, rec.Name, rec.DeactivateAt, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "user", rec.ID, oldRec.Identifiers, rec.Identifiers)

				enqueueInsertChange(batch, "user", revID, rec.ID, diff)

			}

			if err := catbird.EnqueueSend(batch, bbl.UserChangedTopic, bbl.RecordChangedPayload{ID: rec.ID, Rev: revID}, catbird.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.SaveOrganization:
			rec := a.Organization

			// validate
			if err := rec.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			var oldRec *bbl.Organization

			if rec.ID != "" {
				oldRec, err = getOrganization(ctx, tx, rec.ID)
				if err != nil && !errors.Is(err, bbl.ErrNotFound) {
					return fmt.Errorf("AddRev: %s", err)
				}
			}

			// conflict detection
			if oldRec != nil && a.MatchVersion && rec.Version != oldRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, rec.Version, oldRec.Version)
			}

			if oldRec != nil {
				rec.ID = oldRec.ID
			} else {
				rec.ID = bbl.NewID()
			}

			// lookup related organization IDs by identifier
			for i, rel := range rec.Rels {
				if scheme, val, ok := strings.Cut(rel.OrganizationID, ":"); ok {
					id, err := getIDByIdentifier(ctx, r.conn, "organization", "organizations", scheme, val)
					if err != nil {
						return fmt.Errorf("AddRev: %w", err)
					}
					rec.Rels[i].OrganizationID = id
				}
			}

			jsonAttrs, err := json.Marshal(rec.OrganizationAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			if oldRec == nil {
				batch.Queue(`
					INSERT INTO bbl_organizations (id, kind, attrs, version, created_by_id, updated_by_id)
					VALUES ($1, $2, $3, 1, nullif($4, '')::uuid, nullif($5, '')::uuid);`,
					rec.ID, rec.Kind, jsonAttrs, rev.UserID, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "organization", rec.ID, nil, rec.Identifiers)

				enqueueInsertChange(batch, "organization", revID, rec.ID, rec.Diff(&bbl.Organization{}))
			} else {
				// only update if there are changes
				diff := rec.Diff(oldRec)
				if len(diff) == 0 {
					continue
				}

				batch.Queue(`
					UPDATE bbl_organizations
					SET kind = $2,
						attrs = $3,
						version = version + 1,
						updated_at = transaction_timestamp(),
						updated_by_id = nullif($4, '')::uuid
					WHERE id = $1;`,
					rec.ID, rec.Kind, jsonAttrs, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "organization", rec.ID, oldRec.Identifiers, rec.Identifiers)

				enqueueInsertChange(batch, "organization", revID, rec.ID, diff)
			}

			// upsert rels
			if oldRec != nil && len(oldRec.Rels) > len(rec.Rels) {
				batch.Queue(`
						DELETE FROM bbl_organization_rels
						WHERE organization_id = $1 AND idx >= $2;`,
					rec.ID, len(rec.Rels),
				)
			}
			for i, rel := range rec.Rels {
				if oldRec != nil && i < len(oldRec.Rels) {
					if oldRec.Rels[i].Kind == rel.Kind && oldRec.Rels[i].OrganizationID == rel.OrganizationID {
						continue
					}
					batch.Queue(`
							UPDATE bbl_organization_rels
							SET kind = $3,
							    rel_organization_id = $4,
							WHERE organization_id = $1 AND idx = $2;`,
						rec.ID, i, rel.Kind, rel.OrganizationID,
					)
				} else {
					batch.Queue(`
							INSERT INTO bbl_organization_rels (organization_id, idx, kind, rel_organization_id)
							VALUES ($1, $2, $3, $4);`,
						rec.ID, i, rel.Kind, rel.OrganizationID,
					)
				}
			}

			if err := catbird.EnqueueSend(batch, bbl.OrganizationChangedTopic, bbl.RecordChangedPayload{ID: rec.ID, Rev: revID}, catbird.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.SavePerson:
			rec := a.Person

			// validate
			if err := rec.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			var oldRec *bbl.Person

			if rec.ID != "" {
				oldRec, err = getPerson(ctx, tx, rec.ID)
				if err != nil && !errors.Is(err, bbl.ErrNotFound) {
					return fmt.Errorf("AddRev: %s", err)
				}
			}

			// conflict detection
			if oldRec != nil && a.MatchVersion && rec.Version != oldRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, rec.Version, oldRec.Version)
			}

			if oldRec != nil {
				rec.ID = oldRec.ID
			} else {
				rec.ID = bbl.NewID()
			}

			jsonAttrs, err := json.Marshal(rec.PersonAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			if oldRec == nil {
				batch.Queue(`
					INSERT INTO bbl_people (id, attrs, version, created_by_id, updated_by_id)
					VALUES ($1, $2, 1, nullif($3, '')::uuid, nullif($4, '')::uuid);`,
					rec.ID, jsonAttrs, rev.UserID, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "person", rec.ID, nil, rec.Identifiers)

				enqueueInsertChange(batch, "person", revID, rec.ID, rec.Diff(&bbl.Person{}))
			} else {
				// only update if there are changes
				diff := rec.Diff(oldRec)
				if len(diff) == 0 {
					continue
				}

				batch.Queue(`
					UPDATE bbl_people
					SET attrs = $2,
						version = version + 1,
						updated_at = transaction_timestamp(),
						updated_by_id = nullif($3, '')::uuid
					WHERE id = $1;`,
					rec.ID, jsonAttrs, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "person", rec.ID, oldRec.Identifiers, rec.Identifiers)

				enqueueInsertChange(batch, "person", revID, rec.ID, diff)
			}

			if err := catbird.EnqueueSend(batch, bbl.PersonChangedTopic, bbl.RecordChangedPayload{ID: rec.ID, Rev: revID}, catbird.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.SaveProject:
			rec := a.Project

			// validate
			if err := rec.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			var oldRec *bbl.Project

			if rec.ID != "" {
				oldRec, err = getProject(ctx, tx, rec.ID)
				if err != nil && !errors.Is(err, bbl.ErrNotFound) {
					return fmt.Errorf("AddRev: %s", err)
				}
			}

			// conflict detection
			if oldRec != nil && a.MatchVersion && rec.Version != oldRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, rec.Version, oldRec.Version)
			}

			if oldRec != nil {
				rec.ID = oldRec.ID
			} else {
				rec.ID = bbl.NewID()
			}

			jsonAttrs, err := json.Marshal(rec.ProjectAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			if oldRec == nil {
				batch.Queue(`
					INSERT INTO bbl_projects (id, attrs, version, created_by_id, updated_by_id)
					VALUES ($1, $2, 1, nullif($3, '')::uuid, nullif($4, '')::uuid);`,
					rec.ID, jsonAttrs, rev.UserID, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "project", rec.ID, nil, rec.Identifiers)

				enqueueInsertChange(batch, "project", revID, rec.ID, rec.Diff(&bbl.Project{}))
			} else {
				// only update if there are changes
				diff := rec.Diff(oldRec)
				if len(diff) == 0 {
					continue
				}

				batch.Queue(`
					UPDATE bbl_projects
					SET attrs = $2,
						version = version + 1,
						updated_at = transaction_timestamp(),
						updated_by_id = nullif($3, '')::uuid
					WHERE id = $1;`,
					rec.ID, jsonAttrs, rev.UserID,
				)

				enqueueUpsertIdentifiers(batch, "project", rec.ID, oldRec.Identifiers, rec.Identifiers)

				enqueueInsertChange(batch, "project", revID, rec.ID, diff)
			}

			if err := catbird.EnqueueSend(batch, bbl.ProjectChangedTopic, bbl.RecordChangedPayload{ID: rec.ID, Rev: revID}, catbird.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.SaveWork:
			rec := a.Work

			// validate
			if err := rec.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			var oldRec *bbl.Work

			if rec.ID != "" {
				oldRec, err = getWork(ctx, tx, rec.ID)
				if err != nil && !errors.Is(err, bbl.ErrNotFound) {
					return fmt.Errorf("AddRev: %s", err)
				}
			}

			// conflict detection
			if oldRec != nil && a.MatchVersion && rec.Version != oldRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, rec.Version, oldRec.Version)
			}

			if err := saveWork(ctx, tx, batch, revID, rev.UserID, rec, oldRec); err != nil {
				return err
			}
		case *bbl.ChangeWork:
			oldRec, err := getWork(ctx, tx, a.WorkID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			rec, err := oldRec.Clone()
			if err != nil {
				return err
			}

			for _, c := range a.Changes {
				if err := c.Apply(rec); err != nil {
					return err
				}
			}

			if err := saveWork(ctx, tx, batch, revID, rev.UserID, rec, oldRec); err != nil {
				return err
			}
		default:
			return errors.New("AddRev: unknown action")
		}
	}

	// only commit if the rev caused database changes
	if len(batch.QueuedQueries) > 1 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("AddRev: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("AddRev: %w", err)
		}
	}

	return nil
}

func enqueueUpsertIdentifiers(batch *pgx.Batch, t, id string, old, new []bbl.Code) {
	if len(old) > len(new) {
		batch.Queue(`
			DELETE FROM bbl_`+t+`_identifiers
			WHERE `+t+`_id = $1 AND idx >= $2;`,
			id, len(new),
		)
	}
	for i, iden := range new {
		if i < len(old) {
			if old[i].Scheme == iden.Scheme && old[i].Val == iden.Val {
				continue
			}
			batch.Queue(`
				UPDATE bbl_`+t+`_identifiers
				SET scheme = $3,
					val = $4
				WHERE `+t+`_id = $1 AND idx = $2;`,
				id, i, iden.Scheme, iden.Val,
			)
		} else {
			batch.Queue(`
				INSERT INTO bbl_`+t+`_identifiers (`+t+`_id, idx, scheme, val)
				VALUES ($1, $2, $3, $4);`,
				id, i, iden.Scheme, iden.Val,
			)
		}
	}
}

func enqueueInsertChange(batch *pgx.Batch, t, revID, id string, diff any) error {
	jsonDiff, err := json.Marshal(diff)
	if err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}
	batch.Queue(`
		INSERT INTO bbl_changes (rev_id, `+t+`_id, diff)
		VALUES ($1, $2, $3);`,
		revID, id, jsonDiff,
	)
	return nil
}

func saveWork(ctx context.Context, conn Conn, batch *pgx.Batch, revID, userID string, rec, oldRec *bbl.Work) error {
	if oldRec != nil {
		rec.ID = oldRec.ID
	} else {
		rec.ID = bbl.NewID()
	}

	// lookup contributor IDs by identifier
	for i, con := range rec.Contributors {
		if scheme, val, ok := strings.Cut(con.PersonID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "person", "people", scheme, val)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
			rec.Contributors[i].PersonID = id
		}
	}

	// lookup related work IDs by identifier
	for i, rel := range rec.Rels {
		if scheme, val, ok := strings.Cut(rel.WorkID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "work", "works", scheme, val)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
			rec.Rels[i].WorkID = id
		}
	}

	jsonAttrs, err := json.Marshal(rec.WorkAttrs)
	if err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	if oldRec == nil {
		batch.Queue(`
			INSERT INTO bbl_works (id, kind, subkind, status, attrs, version, created_by_id, updated_by_id)
			VALUES ($1, $2, nullif($3, ''), $4, $5, 1, nullif($6, '')::uuid, nullif($7, '')::uuid);`,
			rec.ID, rec.Kind, rec.Subkind, rec.Status, jsonAttrs, userID, userID,
		)

		enqueueUpsertIdentifiers(batch, "work", rec.ID, nil, rec.Identifiers)

		enqueueInsertChange(batch, "work", revID, rec.ID, rec.Diff(&bbl.Work{}))
	} else {
		// only update if there are changes
		diff := rec.Diff(oldRec)
		if bbl.IsZero(*diff) {
			return nil
		}

		batch.Queue(`
			UPDATE bbl_works
			SET kind = $2,
				subkind = nullif($3, ''),
				status = $4,
				attrs = $5,
				version = version + 1,
				updated_at = transaction_timestamp(),
				updated_by_id = nullif($6, '')::uuid
			WHERE id = $1;`,
			rec.ID, rec.Kind, rec.Subkind, rec.Status, jsonAttrs, userID,
		)

		enqueueUpsertIdentifiers(batch, "work", rec.ID, oldRec.Identifiers, rec.Identifiers)

		enqueueInsertChange(batch, "work", revID, rec.ID, diff)
	}

	// upsert contributors
	if oldRec != nil && len(oldRec.Contributors) > len(rec.Contributors) {
		batch.Queue(`
			DELETE FROM bbl_work_contributors
			WHERE work_id = $1 AND idx >= $2;`,
			rec.ID, len(rec.Contributors),
		)
	}
	for i, con := range rec.Contributors {
		jsonAttrs, err := json.Marshal(con.WorkContributorAttrs)
		if err != nil {
			return fmt.Errorf("AddRev: %w", err)
		}

		// TODO only update if different
		if oldRec != nil && i < len(oldRec.Contributors) {
			batch.Queue(`
				UPDATE bbl_work_contributors
				SET attrs = $3,
					person_id = nullif($4, '')::uuid
				WHERE work_id = $1 and idx = $2;`,
				rec.ID, i, jsonAttrs, con.PersonID,
			)
		} else {
			batch.Queue(`
				INSERT INTO bbl_work_contributors (work_id, idx, attrs, person_id)
				VALUES ($1, $2, $3, nullif($4, '')::uuid);`,
				rec.ID, i, jsonAttrs, con.PersonID,
			)
		}
	}

	// upsert files
	if oldRec != nil && len(oldRec.Files) > len(rec.Files) {
		batch.Queue(`
			DELETE FROM bbl_work_files
			WHERE work_id = $1 AND idx >= $2;`,
			rec.ID, len(rec.Files),
		)
	}
	for i, f := range rec.Files {
		// TODO only update if different
		if oldRec != nil && i < len(oldRec.Files) {
			batch.Queue(`
				UPDATE bbl_work_files
				SET object_id = $3,
					name = $4,
					content_type = $5,
					size = $6
				WHERE work_id = $1 AND idx = $2;`,
				rec.ID, i, f.ObjectID, f.Name, f.ContentType, f.Size,
			)
		} else {
			batch.Queue(`
				INSERT INTO bbl_work_files (work_id, idx, object_id, name, content_type, size)
				VALUES ($1, $2, $3, $4, $5, $6);`,
				rec.ID, i, f.ObjectID, f.Name, f.ContentType, f.Size,
			)
		}
	}

	if oldRec != nil && len(oldRec.Rels) > len(rec.Rels) {
		batch.Queue(`
			DELETE FROM bbl_work_rels
			WHERE work_id = $1 AND idx >= $2;`,
			rec.ID, len(rec.Rels),
		)
	}
	for i, rel := range rec.Rels {
		if oldRec.Rels[i].Kind == rel.Kind && oldRec.Rels[i].WorkID == rel.WorkID {
			continue
		}
		if oldRec != nil && i < len(oldRec.Rels) {
			batch.Queue(`
				UPDATE bbl_work_rels
				SET kind = $3,
					rel_work_id = $4
				WHERE work_id = $1 AND idx = $2;`,
				rec.ID, i, rel.Kind, rel.WorkID,
			)
		} else {
			batch.Queue(`
				INSERT INTO bbl_work_rels (work_id, idx, kind, rel_work_id)
				VALUES ($1, $2, $3, $4);`,
				rec.ID, i, rel.Kind, rel.WorkID,
			)
		}
	}

	if err := catbird.EnqueueSend(batch, bbl.WorkChangedTopic, bbl.RecordChangedPayload{ID: rec.ID, Rev: revID}, catbird.SendOpts{}); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	return nil
}

func getIDByIdentifier(ctx context.Context, conn Conn, name, pluralName, scheme, val string) (string, error) {
	// query will fail if identifier is not unique
	q := `SELECT id
		  FROM bbl_` + pluralName + `
		  WHERE id = (SELECT ` + name + `_id
				      FROM bbl_` + name + `_identifiers
				      WHERE scheme = $1 AND val = $2);`

	var id string

	err := conn.QueryRow(ctx, q, scheme, val).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = bbl.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "21000" { // cardinality_violation (mare than one row returned from subquery)
			err = bbl.ErrNotUnique
		}
	}

	return id, err
}
