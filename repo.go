package bbl

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/ugent-library/tonga"
	"go.breu.io/ulid"

	_ "github.com/ugent-library/bbl/migrations"
)

var ErrNotFound = errors.New("not found")

//go:embed migrations/*.sql
var migrationsFS embed.FS

type pgxConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

type Repo struct {
	conn *pgxpool.Pool
	mq   *tonga.Client
}

func NewRepo(ctx context.Context, conn *pgxpool.Pool) (*Repo, error) {
	mq := tonga.New(conn)

	r := &Repo{
		conn: conn,
		mq:   mq,
	}
	return r, nil
}

func (r *Repo) MigrateUp(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db := stdlib.OpenDBFromPool(r.conn)
	defer db.Close()

	return goose.UpContext(ctx, db, "migrations")
}

func (r *Repo) MigrateDown(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	db := stdlib.OpenDBFromPool(r.conn)
	defer db.Close()

	return goose.ResetContext(ctx, db, "migrations")
}

func (r *Repo) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	return getOrganization(ctx, r.conn, id)
}

func (r *Repo) GetPerson(ctx context.Context, id string) (*Person, error) {
	return getPerson(ctx, r.conn, id)
}

func (r *Repo) GetProject(ctx context.Context, id string) (*Project, error) {
	return getProject(ctx, r.conn, id)
}

func (r *Repo) GetWork(ctx context.Context, id string) (*Work, error) {
	return getWork(ctx, r.conn, id)
}

func (r *Repo) Listen(ctx context.Context, queue, topic string, hideFor time.Duration) iter.Seq[Msg] {
	// TODO make channel opts configurable
	if err := r.mq.CreateChannel(ctx, queue, topic, tonga.ChannelOpts{}); err != nil {
		// TODO error handling
		log.Printf("listen: %s", err)
		return nil
	}

	return func(yield func(Msg) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// TODO make quantity configurable
				msgs, err := r.mq.Read(ctx, queue, 10, hideFor)
				if err != nil {
					// TODO error handling
					log.Printf("listen: %s", err)
					return
				}

				for _, m := range msgs {
					msg := Msg{
						queue:     queue,
						id:        m.ID,
						Topic:     m.Topic,
						Body:      m.Body,
						CreatedAt: m.CreatedAt,
					}
					if ok := yield(msg); !ok {
						return
					}
				}

				if len(msgs) < 10 {
					// TODO make backoff configurable
					time.Sleep(1 * time.Second)
				}
			}
		}
	}
}

func (r *Repo) Ack(ctx context.Context, msg Msg) error {
	if _, err := r.mq.Delete(ctx, msg.queue, msg.id); err != nil {
		return err
	}
	return nil
}

func (r *Repo) AddRev(ctx context.Context, rev *Rev) error {
	revID := r.NewID()

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("AddRev: %s", err)
	}
	defer tx.Rollback(ctx)

	mq := tonga.New(tx)

	batch := &pgx.Batch{}

	batch.Queue(`
		insert into bbl_revs (id)
		values ($1);`,
		revID,
	)

	for _, action := range rev.actions {
		switch a := action.(type) {
		case *CreateOrganization:
			if a.Organization.ID == "" {
				a.Organization.ID = r.NewID()
			}

			if err := lookupOrganizationRels(ctx, tx, a.Organization.Rels); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Organization.Diff(&Organization{})

			jsonAttrs, err := json.Marshal(a.Organization.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_organizations (id, kind, source, source_id, attrs)
				values ($1, $2, nullif($3, ''), nullif($4, ''), $5);`,
				a.Organization.ID, a.Organization.Kind, a.Organization.Source, a.Organization.SourceID, jsonAttrs,
			)
			for i, rel := range a.Organization.Rels {
				batch.Queue(`
					insert into bbl_organizations_rels (id, kind, organization_id, rel_organization_id, idx)
					values ($1, $2, $3, $4, $5);`,
					r.NewID(), rel.Kind, a.Organization.ID, rel.OrganizationID, i,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "organization.create", a.Organization.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *UpdateOrganization:
			currentRec, err := getOrganization(ctx, tx, a.Organization.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			if err := lookupOrganizationRels(ctx, tx, a.Organization.Rels); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Organization.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Organization.Attrs)
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
					source = nullif($3, ''),
					source_id = nullif($4, ''),
				    attrs = $5,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Organization.ID, a.Organization.Kind, a.Organization.Source, a.Organization.SourceID, jsonAttrs,
			)

			if _, ok := diff["rels"]; ok {
				for _, currentRel := range currentRec.Rels {
					var found bool
					for _, rel := range a.Organization.Rels {
						if rel.ID == currentRel.ID {
							found = true
							break
						}
					}
					if !found {
						batch.Queue(`
							delete from bbl_organizations_rels
							where id = $1;`,
							currentRel.ID,
						)
					}
				}

				for i, rel := range a.Organization.Rels {
					var found bool
					for _, currentRel := range currentRec.Rels {
						if currentRel.ID == rel.ID {
							found = true
							break
						}
					}
					if found {
						batch.Queue(`
							update bbl_organizations_rels
							set kind = $2,
							    rel_organization_id = $3,
								idx = $4
							where id = $1;`,
							rel.ID, rel.Kind, rel.OrganizationID, i,
						)
					} else {
						batch.Queue(`
							insert into bbl_organizations_rels (id, kind, organization_id, rel_organization_id, idx)
							values ($1, $2, $3, $4, $5);`,
							r.NewID(), rel.Kind, a.Organization.ID, rel.OrganizationID, i,
						)
					}
				}
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "organization.update", a.Organization.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *CreatePerson:
			if a.Person.ID == "" {
				a.Person.ID = r.NewID()
			}

			diff := a.Person.Diff(&Person{})

			jsonAttrs, err := json.Marshal(a.Person.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_people (id, source, source_id, attrs)
				values ($1, nullif($2, ''), nullif($3, ''), $4);`,
				a.Person.ID, a.Person.Source, a.Person.SourceID, jsonAttrs,
			)
			batch.Queue(`
				insert into bbl_changes (rev_id, person_id, diff)
				values ($1, $2, $3);`,
				revID, a.Person.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "person.create", a.Person.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *UpdatePerson:
			currentRec, err := getPerson(ctx, tx, a.Person.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Person.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Person.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				update bbl_people
				set source = nullif($2, ''),
				    source_id = nullif($3, ''),
				    attrs = $4,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Person.ID, a.Person.Source, a.Person.SourceID, jsonAttrs,
			)

			batch.Queue(`
				insert into bbl_changes (rev_id, person_id, diff)
				values ($1, $2, $3);`,
				revID, a.Person.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "person.update", a.Person.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *CreateProject:
			if a.Project.ID == "" {
				a.Project.ID = r.NewID()
			}

			diff := a.Project.Diff(&Project{})

			jsonAttrs, err := json.Marshal(a.Project.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_projects (id, source, source_id, attrs)
				values ($1, nullif($2, ''), nullif($3, ''), $4);`,
				a.Project.ID, a.Project.Source, a.Project.SourceID, jsonAttrs,
			)
			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)
			if err := mq.Send(ctx, "project.create", a.Project.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *UpdateProject:
			currentRec, err := getProject(ctx, tx, a.Project.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Project.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Project.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				update bbl_projects
				set source = nullif($2, ''),
				    source_id = nullif($3, ''),
				    attrs = $4,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Project.ID, a.Project.Source, a.Project.SourceID, jsonAttrs,
			)

			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "project.update", a.Project.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *CreateWork:
			if a.Work.ID == "" {
				a.Work.ID = r.NewID()
			}

			if err := lookupWorkContributors(ctx, tx, a.Work.Contributors); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Work.Diff(&Work{})

			jsonAttrs, err := json.Marshal(a.Work.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			batch.Queue(`
				insert into bbl_works (id, kind, sub_kind, attrs)
				values ($1, $2, nullif($3, ''), $4);`,
				a.Work.ID, a.Work.Kind, a.Work.SubKind, jsonAttrs,
			)
			for i, con := range a.Work.Contributors {
				jsonAttrs, err := json.Marshal(con.Attrs)
				if err != nil {
					return fmt.Errorf("AddRev: %w", err)
				}

				batch.Queue(`
				insert into bbl_works_contributors (id, work_id, attrs, person_id, idx)
				values ($1, $2, $3, $4, $5);`,
					r.NewID(), a.Work.ID, jsonAttrs, con.PersonID, i,
				)
			}
			for i, rel := range a.Work.Rels {
				batch.Queue(`
					insert into bbl_works_rels (id, kind, work_id, rel_work_id, idx)
					values ($1, $2, $3, $4, $5);`,
					r.NewID(), rel.Kind, a.Work.ID, rel.WorkID, i,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, work_id, diff)
				values ($1, $2, $3);`,
				revID, a.Work.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "work.create", a.Work.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}
		case *UpdateWork:
			currentRec, err := getWork(ctx, tx, a.Work.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			if err := lookupWorkContributors(ctx, tx, a.Work.Contributors); err != nil {
				return fmt.Errorf("AddRev: %w", err)
			}

			diff := a.Work.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Work.Attrs)
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
				    sub_kind = nullif($3, ''),
				    attrs = $4,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Work.ID, a.Work.Kind, a.Work.SubKind, jsonAttrs,
			)

			if _, ok := diff["contributors"]; ok {
				for _, currentCon := range currentRec.Contributors {
					var found bool
					for _, con := range a.Work.Contributors {
						if con.ID == currentCon.ID {
							found = true
							break
						}
					}
					if !found {
						batch.Queue(`
							delete from bbl_works_contributors
							where id = $1;`,
							currentCon.ID,
						)
					}
				}

				for i, con := range a.Work.Contributors {
					jsonAttrs, err := json.Marshal(con.Attrs)
					if err != nil {
						return fmt.Errorf("AddRev: %w", err)
					}

					var found bool
					for _, currentCon := range currentRec.Contributors {
						if currentCon.ID == con.ID {
							found = true
							break
						}
					}
					if found {
						batch.Queue(`
							update bbl_works_contributors
							set attrs = $2,
							    person_id = $3,
								idx = $4,
							where id = $1;`,
							con.ID, jsonAttrs, con.PersonID, i,
						)
					} else {
						batch.Queue(`
							insert into bbl_works_contributors (id, work_id, attrs, person_id, idx)
							values ($1, $2, $3, $4, $5);`,
							r.NewID(), a.Work.ID, jsonAttrs, con.PersonID, i,
						)
					}
				}
			}

			if _, ok := diff["rels"]; ok {
				for _, currentRel := range currentRec.Rels {
					var found bool
					for _, rel := range a.Work.Rels {
						if rel.ID == currentRel.ID {
							found = true
							break
						}
					}
					if !found {
						batch.Queue(`
							delete from bbl_works_rels
							where id = $1;`,
							currentRel.ID,
						)
					}
				}

				for i, rel := range a.Work.Rels {
					var found bool
					for _, currentRel := range currentRec.Rels {
						if currentRel.ID == rel.ID {
							found = true
							break
						}
					}
					if found {
						batch.Queue(`
							update bbl_works_rels
							set kind = $2,
							    rel_organization_id = $3,
								idx = $4
							where id = $1;`,
							rel.ID, rel.Kind, rel.WorkID, i,
						)
					} else {
						batch.Queue(`
							insert into bbl_works_rels (id, kind, work_id, rel_work_id, i)
							values ($1, $2, $3, $4, $5);`,
							r.NewID(), rel.Kind, a.Work.ID, rel.WorkID, i,
						)
					}
				}
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, work_id, diff)
				values ($1, $2, $3);`,
				revID, a.Work.ID, jsonDiff,
			)

			if err := mq.Send(ctx, "work.update", a.Work.ID, tonga.SendOpts{}); err != nil {
				return fmt.Errorf("AddRev: %w", err)
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("AddRev: %w", err)
	}

	return nil
}

func lookupOrganizationRels(ctx context.Context, conn pgxConn, rels []OrganizationRel) error {
	for i, rel := range rels {
		if sourceStr, ok := strings.CutPrefix(rel.OrganizationID, "source:"); ok {
			if source, sourceID, ok := strings.Cut(sourceStr, ":"); ok {
				id, err := getOrganizationIDBySource(ctx, conn, source, sourceID)
				if err != nil {
					return err
				}
				rels[i].OrganizationID = id
			}
		}
	}
	return nil
}

func lookupWorkContributors(ctx context.Context, conn pgxConn, contributors []WorkContributor) error {
	for i, con := range contributors {
		if sourceStr, ok := strings.CutPrefix(con.PersonID, "source:"); ok {
			if source, sourceID, ok := strings.Cut(sourceStr, ":"); ok {
				id, err := getPersonIDBySource(ctx, conn, source, sourceID)
				if err != nil {
					return err
				}
				contributors[i].PersonID = id
			}
		}
	}
	return nil
}

func (r *Repo) NewID() string {
	return ulid.Make().UUIDString()
}

func getOrganizationIDBySource(ctx context.Context, conn pgxConn, source, sourceID string) (string, error) {
	q := `
		select id
		from bbl_organizations
		where source = $1 and source_id = $2;`

	var id string

	err := conn.QueryRow(ctx, q, source, sourceID).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return id, err
}

func getOrganization(ctx context.Context, conn pgxConn, id string) (*Organization, error) {
	rec, err := getOrganizationBy(ctx, conn, "id = $1", id)
	if err != nil {
		err = fmt.Errorf("GetOrganization %s: %w", id, err)
	}
	return rec, err
}

func getOrganizationBy(ctx context.Context, conn pgxConn, where string, args ...any) (*Organization, error) {
	q := `
		select id, kind, coalesce(source, ''), coalesce(source_id, ''), attrs, rels, created_at, updated_at
		from bbl_organizations_view
		where ` + where + `;`

	rec, err := scanOrganization(conn.QueryRow(ctx, q, args...))
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return rec, err
}

func scanOrganization(row pgx.Row) (*Organization, error) {
	var rec Organization
	var rawAttrs json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rec.Source, &rec.SourceID, &rawAttrs, &rawRels, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}

func getPersonIDBySource(ctx context.Context, conn pgxConn, source, sourceID string) (string, error) {
	q := `
		select id
		from bbl_people
		where source = $1 and source_id = $2;`

	var id string

	err := conn.QueryRow(ctx, q, source, sourceID).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return id, err
}

func getPerson(ctx context.Context, conn pgxConn, id string) (*Person, error) {
	rec, err := getPersonBy(ctx, conn, "id = $1", id)
	if err != nil {
		err = fmt.Errorf("GetPerson %s: %w", id, err)
	}
	return rec, err
}

func getPersonBy(ctx context.Context, conn pgxConn, where string, args ...any) (*Person, error) {
	q := `
		select id, coalesce(source, ''), coalesce(source_id, ''), attrs, created_at, updated_at
		from bbl_people
		where ` + where + `;`

	rec, err := scanPerson(conn.QueryRow(ctx, q, args...))
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return rec, err
}

func scanPerson(row pgx.Row) (*Person, error) {
	var rec Person
	var rawAttrs json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Source, &rec.SourceID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	return &rec, nil
}

// func getProjectIDBySource(ctx context.Context, conn pgxConn, source, sourceID string) (string, error) {
// 	q := `
// 		select id
// 		from bbl_projects
// 		where source = $1 and source_id = $2;`

// 	var id string

// 	err := conn.QueryRow(ctx, q, source, sourceID).Scan(&id)
// 	if err == pgx.ErrNoRows {
// 		err = ErrNotFound
// 	}

// 	return id, err
// }

func getProject(ctx context.Context, conn pgxConn, id string) (*Project, error) {
	rec, err := getProjectBy(ctx, conn, "id = $1", id)
	if err != nil {
		err = fmt.Errorf("GetProject %s: %w", id, err)
	}
	return rec, err
}

func getProjectBy(ctx context.Context, conn pgxConn, where string, args ...any) (*Project, error) {
	q := `
		select id, coalesce(source, ''), coalesce(source_id, ''), attrs, created_at, updated_at
		from bbl_projects
		where ` + where + `;`

	rec, err := scanProject(conn.QueryRow(ctx, q, args...))
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return rec, err
}

func scanProject(row pgx.Row) (*Project, error) {
	var rec Project
	var rawAttrs json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Source, &rec.SourceID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	return &rec, nil
}

func getWork(ctx context.Context, conn pgxConn, id string) (*Work, error) {
	rec, err := getWorkBy(ctx, conn, "id = $1", id)
	if err != nil {
		err = fmt.Errorf("GetWork %s: %w", id, err)
	}
	return rec, err
}

func getWorkBy(ctx context.Context, conn pgxConn, where string, args ...any) (*Work, error) {
	q := `
		select id, kind, coalesce(sub_kind, ''), attrs, contributors, rels, created_at, updated_at
		from bbl_works_view
		where ` + where + `;`

	rec, err := scanWork(conn.QueryRow(ctx, q, args...))
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return rec, err
}

func scanWork(row pgx.Row) (*Work, error) {
	var rec Work
	var rawAttrs json.RawMessage
	var rawContributors json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rec.SubKind, &rawAttrs, &rawContributors, &rawRels, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawContributors != nil {
		if err := json.Unmarshal(rawContributors, &rec.Contributors); err != nil {
			return nil, err
		}
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, err
		}
	}

	if err := LoadWorkProfile(&rec); err != nil {
		return nil, err
	}

	return &rec, nil
}
