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

func (r *Repo) NewID() string {
	return ulid.Make().UUIDString()
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
				insert into bbl_organizations (id, kind, attrs)
				values ($1, $2, $3);`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs,
			)
			for i, iden := range a.Organization.Identifiers {
				batch.Queue(`
					insert into bbl_organizations_identifiers (organization_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Organization.ID, i, iden.Scheme, iden.Val,
				)
			}
			for i, rel := range a.Organization.Rels {
				batch.Queue(`
					insert into bbl_organizations_rels (organization_id, idx, kind, rel_organization_id)
					values ($1, $2, $3, $4);`,
					a.Organization.ID, i, rel.Kind, rel.OrganizationID,
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
				    attrs = $3,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs,
			)

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "organization", "organizations", a.Organization.ID, currentRec.Identifiers, a.Organization.Identifiers)
			}

			if _, ok := diff["rels"]; ok {
				if len(currentRec.Rels) > len(a.Organization.Rels) {
					batch.Queue(`
						delete from bbl_organizations_rels
						where organization_id = $1 and idx >= $2;`,
						a.Organization.ID, len(a.Organization.Rels),
					)
				}
				for i, rel := range a.Organization.Rels {
					// TODO only update if different
					if i < len(currentRec.Rels) {
						batch.Queue(`
							update bbl_organizations_rels
							set kind = $3,
							    rel_organization_id = $4,
							where organization_id = $1 and idx = $2;`,
							a.Organization.ID, i, rel.Kind, rel.OrganizationID,
						)
					} else {
						batch.Queue(`
							insert into bbl_organizations_rels (organization_id, idx, kind, rel_organization_id)
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
				insert into bbl_people (id, attrs)
				values ($1, $2);`,
				a.Person.ID, jsonAttrs,
			)
			for i, iden := range a.Person.Identifiers {
				batch.Queue(`
					insert into bbl_people_identifiers (person_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Person.ID, i, iden.Scheme, iden.Val,
				)
			}
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
				set attrs = $2,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Person.ID, jsonAttrs,
			)

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "person", "people", a.Person.ID, currentRec.Identifiers, a.Person.Identifiers)
			}

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
				insert into bbl_projects (id, attrs)
				values ($1, $2);`,
				a.Project.ID, jsonAttrs,
			)
			for i, iden := range a.Project.Identifiers {
				batch.Queue(`
					insert into bbl_projects_identifiers (project_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Project.ID, i, iden.Scheme, iden.Val,
				)
			}
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
				set attrs = $2,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Project.ID, jsonAttrs,
			)

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "project", "projects", a.Project.ID, currentRec.Identifiers, a.Project.Identifiers)
			}

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
			for i, iden := range a.Work.Identifiers {
				batch.Queue(`
					insert into bbl_works_identifiers (work_id, idx, scheme, val, uniq)
					values ($1, $2, $3, $4, true);`,
					a.Work.ID, i, iden.Scheme, iden.Val,
				)
			}
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
					insert into bbl_works_rels (work_id, idx, kind, rel_work_id)
					values ($1, $2, $3, $4);`,
					a.Work.ID, i, rel.Kind, rel.WorkID,
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

			if _, ok := diff["identifiers"]; ok {
				queueUpdateIdentifiersQueries(batch, "work", "works", a.Work.ID, currentRec.Identifiers, a.Work.Identifiers)
			}

			if _, ok := diff["contributors"]; ok {
				if len(currentRec.Contributors) > len(a.Work.Contributors) {
					batch.Queue(`
						delete from bbl_works_contributors
						where work_id = $1 and idx >= $2;`,
						a.Work.ID, len(a.Work.Contributors),
					)
				}
				for i, con := range a.Work.Contributors {
					jsonAttrs, err := json.Marshal(con.Attrs)
					if err != nil {
						return fmt.Errorf("AddRev: %w", err)
					}

					// TODO only update if different
					if i < len(currentRec.Contributors) {
						batch.Queue(`
							update bbl_works_contributors
							set attrs = $3,
							    person_id = $4,
							where work_id = $1 and idx = $2;`,
							a.Work.ID, i, jsonAttrs, con.PersonID,
						)
					} else {
						batch.Queue(`
							insert into bbl_works_contributors (work_id, idx, attrs, person_id)
							values ($1, $2, $3, $4);`,
							a.Work.ID, i, jsonAttrs, con.PersonID,
						)
					}
				}
			}

			if _, ok := diff["rels"]; ok {
				if len(currentRec.Rels) > len(a.Work.Rels) {
					batch.Queue(`
						delete from bbl_works_rels
						where work_id = $1 and idx >= $2;`,
						a.Work.ID, len(a.Work.Rels),
					)
				}
				for i, rel := range a.Work.Rels {
					// TODO only update if different
					if i < len(currentRec.Rels) {
						batch.Queue(`
							update bbl_works_rels
							set kind = $3,
							    rel_work_id = $4,
							where work_id = $1 and idx = $2;`,
							a.Work.ID, i, rel.Kind, rel.WorkID,
						)
					} else {
						batch.Queue(`
							insert into bbl_works_rels (work_id, idx, kind, rel_work_id)
							values ($1, $2, $3, $4);`,
							a.Work.ID, i, rel.Kind, rel.WorkID,
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
		if scheme, val, ok := strings.Cut(rel.OrganizationID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "organization", "organizations", scheme, val)
			if err != nil {
				return err
			}
			rels[i].OrganizationID = id
		}
	}
	return nil
}

func lookupWorkContributors(ctx context.Context, conn pgxConn, contributors []WorkContributor) error {
	for i, con := range contributors {
		if scheme, val, ok := strings.Cut(con.PersonID, ":"); ok {
			id, err := getIDByIdentifier(ctx, conn, "person", "people", scheme, val)
			if err != nil {
				return err
			}
			contributors[i].PersonID = id
		}
	}
	return nil
}

func queueUpdateIdentifiersQueries(batch *pgx.Batch, name, pluralName, id string, old, new []Code) {
	if len(old) > len(new) {
		batch.Queue(`
			delete from bbl_`+pluralName+`_identifiers
			where `+name+`_id = $1 and idx >= $2;`,
			id, len(new),
		)
	}
	for i, ident := range new {
		// TODO only update if different
		if i < len(old) {
			batch.Queue(`
				update bbl_`+pluralName+`_identifiers
				set scheme = $3,
					val = $4,
					uniq = true
				where `+name+`_id = $1 and idx = $2;`,
				id, i, ident.Scheme, ident.Val,
			)
		} else {
			batch.Queue(`
				insert into bbl_`+pluralName+`_identifiers (`+name+`_id, idx, scheme, val, uniq)
				values ($1, $2, $3, $4, true);`,
				id, i, ident.Scheme, ident.Val,
			)
		}
	}
}

func getIDByIdentifier(ctx context.Context, conn pgxConn, name, pluralName, scheme, val string) (string, error) {
	q := `
		select ` + name + `_id
		from bbl_` + pluralName + `_identifiers
		where scheme = $1 and val = $2 and uniq = true;`

	var id string

	err := conn.QueryRow(ctx, q, scheme, val).Scan(&id)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}

	return id, err
}

func getOrganization(ctx context.Context, conn pgxConn, id string) (*Organization, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select o.id, o.kind, o.attrs, o.created_at, o.updated_at, o.identifiers, o.rels
			from bbl_organizations_view o, bbl_organizations_identifiers o_i
			where o.id = o_i.organizatons_id and o_i.scheme = $1 and o_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, kind, attrs, created_at, updated_at, identifiers, rels
			from bbl_organizations_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanOrganization(row)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetOrganization %s: %w", id, err)
	}

	return rec, err
}

func scanOrganization(row pgx.Row) (*Organization, error) {
	var rec Organization
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers, &rawRels); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}

func getPerson(ctx context.Context, conn pgxConn, id string) (*Person, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select p.id, p.attrs, p.created_at, p.updated_at, p.identifiers
			from bbl_people_view w, bbl_people_identifiers p_i
			where p.id = p_i.person_id and p_i.scheme = $1 and p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, attrs, created_at, updated_at, identifiers
			from bbl_people_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanPerson(row)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetPerson %s: %w", id, err)
	}

	return rec, err
}

func scanPerson(row pgx.Row) (*Person, error) {
	var rec Person
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}

func getProject(ctx context.Context, conn pgxConn, id string) (*Project, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select p.id, p.attrs, p.created_at, p.updated_at, p.identifiers
			from bbl_projects_view w, bbl_projects_identifiers p_i
			where p.id = p_i.project_id and p_i.scheme = $1 and p_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, attrs, created_at, updated_at, identifiers
			from bbl_projects_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanProject(row)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetProject %s: %w", id, err)
	}

	return rec, err
}

func scanProject(row pgx.Row) (*Project, error) {
	var rec Project
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage

	if err := row.Scan(&rec.ID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
	}

	return &rec, nil
}

func getWork(ctx context.Context, conn pgxConn, id string) (*Work, error) {
	var row pgx.Row
	if scheme, val, ok := strings.Cut(id, ":"); ok {
		row = conn.QueryRow(ctx, `
			select w.id, w.kind, coalesce(w.sub_kind, ''), w.attrs, w.created_at, w.updated_at, w.identifiers, w.contributors, w.rels
			from bbl_works_view w, bbl_works_identifiers w_i
			where w.id = w_i.work_id and w_i.scheme = $1 and w_i.val = $2;`,
			scheme, val,
		)
	} else {
		row = conn.QueryRow(ctx, `
			select id, kind, coalesce(sub_kind, ''), attrs, created_at, updated_at, identifiers, contributors, rels
			from bbl_works_view
			where id = $1;`,
			id,
		)
	}

	rec, err := scanWork(row)
	if err == pgx.ErrNoRows {
		err = ErrNotFound
	}
	if err != nil {
		err = fmt.Errorf("GetWork %s: %w", id, err)
	}

	return rec, err
}

func scanWork(row pgx.Row) (*Work, error) {
	var rec Work
	var rawAttrs json.RawMessage
	var rawIdentifiers json.RawMessage
	var rawContributors json.RawMessage
	var rawRels json.RawMessage

	if err := row.Scan(&rec.ID, &rec.Kind, &rec.SubKind, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt, &rawIdentifiers, &rawContributors, &rawRels); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, err
	}

	if rawIdentifiers != nil {
		if err := json.Unmarshal(rawIdentifiers, &rec.Identifiers); err != nil {
			return nil, err
		}
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
