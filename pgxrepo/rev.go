package pgxrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/tonga"
)

func (r *Repo) AddRev(ctx context.Context, rev *bbl.Rev) error {
	revID := bbl.NewID()

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("AddRev: %s", err)
	}
	defer tx.Rollback(ctx)

	mq := tonga.New(tx)

	riverJobs := []river.InsertManyParams{}

	batch := &pgx.Batch{}

	batch.Queue(`
		insert into bbl_revs (id, user_id)
		values ($1, nullif($2, '')::uuid);`,
		revID, rev.UserID,
	)

	for _, action := range rev.Actions {
		switch a := action.(type) {
		case *bbl.CreateOrganization:
			if a.Organization.ID == "" {
				a.Organization.ID = bbl.NewID()
			}

			if err := lookupOrganizationRels(ctx, tx, a.Organization.Rels); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// validate
			if err := a.Organization.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Organization.Diff(&bbl.Organization{})

			jsonAttrs, err := json.Marshal(a.Organization.OrganizationAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_organizations (id, kind, attrs, version, created_by_id, updated_by_id)
				values ($1, $2, $3, 1, nullif($4, '')::uuid, nullif($5, '')::uuid);`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs, rev.UserID, rev.UserID,
			)
			for i, iden := range a.Organization.Identifiers {
				batch.Queue(`
					insert into bbl_organization_identifiers (organization_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Organization.ID, i, iden.Scheme, iden.Val,
				)
			}
			for i, rel := range a.Organization.Rels {
				batch.Queue(`
					insert into bbl_organization_rels (organization_id, idx, kind, rel_organization_id)
					values ($1, $2, $3, $4);`,
					a.Organization.ID, i, rel.Kind, rel.OrganizationID,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexOrganization{ID: a.Organization.ID}})

			if err := mq.Send(ctx, "organization.create", a.Organization.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.UpdateOrganization:
			currentRec, err := getOrganization(ctx, tx, a.Organization.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// conflict detection
			if a.MatchVersion && a.Organization.Version != currentRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, a.Organization.Version, currentRec.Version)
			}

			if err := lookupOrganizationRels(ctx, tx, a.Organization.Rels); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// validate
			if err := a.Organization.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Organization.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Organization.OrganizationAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				update bbl_organizations
				set kind = $2,
				    attrs = $3,
					version = version + 1,
				    updated_at = transaction_timestamp(),
					updated_by_id = nullif($4, '')::uuid
				where id = $1;`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs, rev.UserID,
			)

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "organization", a.Organization.ID, currentRec.Identifiers, a.Organization.Identifiers)
			}

			if _, ok := diff["rels"]; ok {
				if len(currentRec.Rels) > len(a.Organization.Rels) {
					batch.Queue(`
						delete from bbl_organization_rels
						where organization_id = $1 and idx >= $2;`,
						a.Organization.ID, len(a.Organization.Rels),
					)
				}
				for i, rel := range a.Organization.Rels {
					// TODO only update if different
					if i < len(currentRec.Rels) {
						batch.Queue(`
							update bbl_organization_rels
							set kind = $3,
							    rel_organization_id = $4,
							where organization_id = $1 and idx = $2;`,
							a.Organization.ID, i, rel.Kind, rel.OrganizationID,
						)
					} else {
						batch.Queue(`
							insert into bbl_organization_rels (organization_id, idx, kind, rel_organization_id)
							values ($1, $2, $3, $4);`,
							a.Organization.ID, i, rel.Kind, rel.OrganizationID,
						)
					}
				}
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexOrganization{ID: a.Organization.ID}})

			if err := mq.Send(ctx, "organization.update", a.Organization.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.CreatePerson:
			if a.Person.ID == "" {
				a.Person.ID = bbl.NewID()
			}

			// validate
			if err := a.Person.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Person.Diff(&bbl.Person{})

			jsonAttrs, err := json.Marshal(a.Person.PersonAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_people (id, attrs, version, created_by_id, updated_by_id)
				values ($1, $2, 1, nullif($3, '')::uuid, nullif($4, '')::uuid);`,
				a.Person.ID, jsonAttrs, rev.UserID, rev.UserID,
			)
			for i, iden := range a.Person.Identifiers {
				batch.Queue(`
					insert into bbl_person_identifiers (person_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Person.ID, i, iden.Scheme, iden.Val,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, person_id, diff)
				values ($1, $2, $3);`,
				revID, a.Person.ID, jsonDiff,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexPerson{ID: a.Person.ID}})

			if err := mq.Send(ctx, "person.create", a.Person.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.UpdatePerson:
			currentRec, err := getPerson(ctx, tx, a.Person.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// conflict detection
			if a.MatchVersion && a.Person.Version != currentRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, a.Person.Version, currentRec.Version)
			}

			// validate
			if err := a.Person.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Person.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Person.PersonAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				update bbl_people
				set attrs = $2,
					version = version + 1,
				    updated_at = transaction_timestamp(),
					updated_by_id = nullif($3, '')::uuid
				where id = $1;`,
				a.Person.ID, jsonAttrs, rev.UserID,
			)

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "person", a.Person.ID, currentRec.Identifiers, a.Person.Identifiers)
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, person_id, diff)
				values ($1, $2, $3);`,
				revID, a.Person.ID, jsonDiff,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexPerson{ID: a.Person.ID}})

			if err := mq.Send(ctx, "person.update", a.Person.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.CreateProject:
			if a.Project.ID == "" {
				a.Project.ID = bbl.NewID()
			}

			// validate
			if err := a.Project.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Project.Diff(&bbl.Project{})

			jsonAttrs, err := json.Marshal(a.Project.ProjectAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_projects (id, attrs, version, created_by_id, updated_by_id)
				values ($1, $2, 1, nullif($3, '')::uuid, nullif($4, '')::uuid);`,
				a.Project.ID, jsonAttrs, rev.UserID, rev.UserID,
			)
			for i, iden := range a.Project.Identifiers {
				batch.Queue(`
					insert into bbl_project_identifiers (project_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Project.ID, i, iden.Scheme, iden.Val,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexProject{ID: a.Project.ID}})

			if err := mq.Send(ctx, "project.create", a.Project.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.UpdateProject:
			currentRec, err := getProject(ctx, tx, a.Project.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// conflict detection
			if a.MatchVersion && a.Project.Version != currentRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, a.Project.Version, currentRec.Version)
			}

			// validate
			if err := a.Project.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Project.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Project.ProjectAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				update bbl_projects
				set attrs = $2,
					version = version + 1,
				    updated_at = transaction_timestamp(),
					updated_by_id = nullif($3, '')::uuid
				where id = $1;`,
				a.Project.ID, jsonAttrs, rev.UserID,
			)

			riverJobs = append(riverJobs, river.InsertManyParams{Args: jobs.IndexProject{ID: a.Project.ID}})

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "project", a.Project.ID, currentRec.Identifiers, a.Project.Identifiers)
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "project.update", a.Project.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.CreateWork:
			if a.Work.ID == "" {
				a.Work.ID = bbl.NewID()
			}

			if err := lookupWorkContributors(ctx, tx, a.Work.Contributors); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// validate
			if err := a.Work.Validate(); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Work.Diff(&bbl.Work{})

			jsonAttrs, err := json.Marshal(a.Work.WorkAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_works (id, kind, subkind, status, attrs, version, created_by_id, updated_by_id)
				values ($1, $2, nullif($3, ''), $4, $5, 1, nullif($6, '')::uuid, nullif($7, '')::uuid);`,
				a.Work.ID, a.Work.Kind, a.Work.Subkind, a.Work.Status, jsonAttrs, rev.UserID, rev.UserID,
			)

			if diff.Permissions != nil {
				for _, perm := range a.Work.Permissions {
					batch.Queue(`
						insert into bbl_work_permissions (work_id, kind, user_id)
						values ($1, $2, $3);`,
						a.Work.ID, perm.Kind, perm.UserID,
					)
				}
			}

			for i, iden := range a.Work.Identifiers {
				batch.Queue(`
					insert into bbl_work_identifiers (work_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Work.ID, i, iden.Scheme, iden.Val,
				)
			}
			for i, con := range a.Work.Contributors {
				jsonAttrs, err := json.Marshal(con.WorkContributorAttrs)
				if err != nil {
					return fmt.Errorf("AddRev: %w", err)
				}

				batch.Queue(`
				insert into bbl_work_contributors (work_id, idx, attrs, person_id)
				values ($1, $2, $3, nullif($4, '')::uuid);`,
					a.Work.ID, i, jsonAttrs, con.PersonID,
				)
			}
			for i, f := range a.Work.Files {
				batch.Queue(`
					insert into bbl_work_files (work_id, idx, object_id, name, content_type, size)
					values ($1, $2, $3, $4, $5, $6);`,
					a.Work.ID, i, f.ObjectID, f.Name, f.ContentType, f.Size,
				)
			}
			for i, rel := range a.Work.Rels {
				batch.Queue(`
					insert into bbl_work_rels (work_id, idx, kind, rel_work_id)
					values ($1, $2, $3, $4);`,
					a.Work.ID, i, rel.Kind, rel.WorkID,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, work_id, diff)
				values ($1, $2, $3);`,
				revID, a.Work.ID, jsonDiff,
			)

			riverJobs = append(riverJobs,
				river.InsertManyParams{Args: jobs.IndexWork{ID: a.Work.ID}},
				river.InsertManyParams{Args: jobs.AddWorkRepresentations{ID: a.Work.ID}},
			)

			if err := mq.Send(ctx, "work.create", a.Work.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *bbl.UpdateWork:
			currentRec, err := getWork(ctx, tx, a.Work.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// conflict detection
			if a.MatchVersion && a.Work.Version != currentRec.Version {
				return fmt.Errorf("AddRev: %w: got %d, expected %d", bbl.ErrConflict, a.Work.Version, currentRec.Version)
			}

			if err := updateWork(ctx, tx, batch, mq, &riverJobs, revID, rev.UserID, a.Work, currentRec); err != nil {
				return err
			}
		case *bbl.ChangeWork:
			currentRec, err := getWork(ctx, tx, a.WorkID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			rec, err := currentRec.Clone()
			if err != nil {
				return err
			}

			for _, c := range a.Changes {
				if err := c.Apply(rec); err != nil {
					return err
				}
			}

			if err := updateWork(ctx, tx, batch, mq, &riverJobs, revID, rev.UserID, rec, currentRec); err != nil {
				return err
			}
		default:
			return errors.New("AddRev: unknown action")
		}
	}

	res := tx.SendBatch(ctx, batch)
	defer res.Close()

	for i := 0; i < batch.Len(); i++ {
		if _, err := res.Exec(); err != nil {
			return fmt.Errorf("AddRev: %w: %s", err, batch.QueuedQueries[i].SQL)
		}
	}

	if err := res.Close(); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	if len(riverJobs) > 0 {
		_, err := r.riverClient.InsertManyTx(ctx, tx, riverJobs)
		if err != nil {
			return fmt.Errorf("AddRev: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	return nil
}

func lookupOrganizationRels(ctx context.Context, conn pgxConn, rels []bbl.OrganizationRel) error {
	for i, rel := range rels {
		if scheme, val, ok := strings.Cut(rel.OrganizationID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "organization", scheme, val)
			if err != nil {
				return err
			}
			rels[i].OrganizationID = id
		}
	}
	return nil
}

func lookupWorkContributors(ctx context.Context, conn pgxConn, contributors []bbl.WorkContributor) error {
	for i, con := range contributors {
		if scheme, val, ok := strings.Cut(con.PersonID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "person", scheme, val)
			if err != nil {
				return err
			}
			contributors[i].PersonID = id
		}
	}
	return nil
}

func queueUpdateIdentifiersQueries(batch *pgx.Batch, name, id string, old, new []bbl.Code) {
	if len(old) > len(new) {
		batch.Queue(`
			delete from bbl_`+name+`_identifiers
			where `+name+`_id = $1 and idx >= $2;`,
			id, len(new),
		)
	}
	for i, ident := range new {
		// TODO only update if different
		if i < len(old) {
			batch.Queue(`
				update bbl_`+name+`_identifiers
				set scheme = $3,
					val = $4,
					uniq = true
				where `+name+`_id = $1 and idx = $2;`,
				id, i, ident.Scheme, ident.Val,
			)
		} else {
			batch.Queue(`
				insert into bbl_`+name+`_identifiers (`+name+`_id, idx, scheme, val, uniq)
				values ($1, $2, $3, $4, true);`,
				id, i, ident.Scheme, ident.Val,
			)
		}
	}
}

func updateWork(ctx context.Context, tx pgx.Tx, batch *pgx.Batch, mq *tonga.Client, riverJobs *[]river.InsertManyParams, revID, userID string, rec, currentRec *bbl.Work) error {
	if err := lookupWorkContributors(ctx, tx, rec.Contributors); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	// validate
	if err := rec.Validate(); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	diff := rec.Diff(currentRec)

	if bbl.IsZero(*diff) {
		return nil
	}

	jsonAttrs, err := json.Marshal(rec.WorkAttrs)
	if err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	jsonDiff, err := json.Marshal(diff)
	if err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	batch.Queue(`
		update bbl_works
		set kind = $2,
			subkind = nullif($3, ''),
			status = $4,
			attrs = $5,
			version = version + 1,
			updated_at = transaction_timestamp(),
			updated_by_id = nullif($6, '')::uuid
		where id = $1;`,
		rec.ID, rec.Kind, rec.Subkind, rec.Status, jsonAttrs, userID,
	)

	if diff.Permissions != nil {
		batch.Queue(`
			delete from bbl_work_permissions
			where work_id = $1;`,
			rec.ID,
		)
		for _, perm := range rec.Permissions {
			batch.Queue(`
				insert into bbl_work_permissions (work_id, kind, user_id)
				values ($1, $2, $3);`,
				rec.ID, perm.Kind, perm.UserID,
			)
		}
	}

	if diff.Identifiers != nil {
		queueUpdateIdentifiersQueries(batch, "work", rec.ID, currentRec.Identifiers, rec.Identifiers)
	}

	if diff.Contributors != nil {
		if len(currentRec.Contributors) > len(rec.Contributors) {
			batch.Queue(`
				delete from bbl_work_contributors
				where work_id = $1 and idx >= $2;`,
				rec.ID, len(rec.Contributors),
			)
		}
		for i, con := range rec.Contributors {
			jsonAttrs, err := json.Marshal(con.WorkContributorAttrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			// TODO only update if different
			if i < len(currentRec.Contributors) {
				batch.Queue(`
					update bbl_work_contributors
					set attrs = $3,
						person_id = nullif($4, '')::uuid
					where work_id = $1 and idx = $2;`,
					rec.ID, i, jsonAttrs, con.PersonID,
				)
			} else {
				batch.Queue(`
					insert into bbl_work_contributors (work_id, idx, attrs, person_id)
					values ($1, $2, $3, nullif($4, '')::uuid);`,
					rec.ID, i, jsonAttrs, con.PersonID,
				)
			}
		}
	}

	if diff.Files != nil {
		if len(currentRec.Files) > len(rec.Files) {
			batch.Queue(`
				delete from bbl_work_files
				where work_id = $1 and idx >= $2;`,
				rec.ID, len(rec.Files),
			)
		}
		for i, f := range rec.Files {
			// TODO only update if different
			if i < len(currentRec.Files) {
				batch.Queue(`
					update bbl_work_files
					set object_id = $3,
						name = $4,
						content_type = $5,
						size = $6
					where work_id = $1 and idx = $2;`,
					rec.ID, i, f.ObjectID, f.Name, f.ContentType, f.Size,
				)
			} else {
				batch.Queue(`
					insert into bbl_work_files (work_id, idx, object_id, name, content_type, size)
					values ($1, $2, $3, $4, $5, $6);`,
					rec.ID, i, f.ObjectID, f.Name, f.ContentType, f.Size,
				)
			}
		}
	}

	if diff.Rels != nil {
		if len(currentRec.Rels) > len(rec.Rels) {
			batch.Queue(`
				delete from bbl_work_rels
				where work_id = $1 and idx >= $2;`,
				rec.ID, len(rec.Rels),
			)
		}
		for i, rel := range rec.Rels {
			// TODO only update if different
			if i < len(currentRec.Rels) {
				batch.Queue(`
					update bbl_work_rels
					set kind = $3,
						rel_work_id = $4
					where work_id = $1 and idx = $2;`,
					rec.ID, i, rel.Kind, rel.WorkID,
				)
			} else {
				batch.Queue(`
					insert into bbl_work_rels (work_id, idx, kind, rel_work_id)
					values ($1, $2, $3, $4);`,
					rec.ID, i, rel.Kind, rel.WorkID,
				)
			}
		}
	}

	batch.Queue(`
		insert into bbl_changes (rev_id, work_id, diff)
		values ($1, $2, $3);`,
		revID, rec.ID, jsonDiff,
	)

	// TODO ugly
	*riverJobs = append(*riverJobs,
		river.InsertManyParams{Args: jobs.IndexWork{ID: rec.ID}},
		river.InsertManyParams{Args: jobs.AddWorkRepresentations{ID: rec.ID}},
	)

	if err := mq.Send(ctx, "work.update", rec.ID, tonga.SendOpts{}); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	return nil
}

func getIDByIdentifier(ctx context.Context, conn pgxConn, name, scheme, val string) (string, error) {
	q := `
		select ` + name + `_id
		from bbl_` + name + `_identifiers
		where scheme = $1 and val = $2 and uniq = true;`

	var id string

	err := conn.QueryRow(ctx, q, scheme, val).Scan(&id)
	if err == pgx.ErrNoRows {
		err = bbl.ErrNotFound
	}

	return id, err
}
