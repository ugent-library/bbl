package bbl

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"

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

func NewRepo(conn *pgxpool.Pool) (*Repo, error) {
	r := &Repo{
		conn: conn,
		mq:   tonga.New(conn),
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

func (r *Repo) GetProject(ctx context.Context, id string) (*Project, error) {
	return getProject(ctx, r.conn, id)
}

func (r *Repo) GetWork(ctx context.Context, id string) (*Work, error) {
	return getWork(ctx, r.conn, id)
}

func (r *Repo) AddRev(ctx context.Context, rev *Rev) error {
	revID := r.NewID()

	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("AddRev: %s", err)
	}
	defer tx.Rollback(ctx)

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
			diff := a.Organization.Diff(&Organization{})

			jsonAttrs, err := json.Marshal(a.Organization.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			batch.Queue(`
				insert into bbl_organizations (id, kind, attrs)
				values ($1, $2, $3);`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs,
			)
			for _, rel := range a.Organization.Rels {
				batch.Queue(`
					insert into bbl_organizations_rels (id, kind, organization_id, rel_organization_id)
					values ($1, $2, $3, $4);`,
					r.NewID(), rel.Kind, a.Organization.ID, rel.OrganizationID,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)
		case *UpdateOrganization:
			currentRec, err := getOrganization(ctx, tx, a.Organization.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			diff := a.Organization.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Organization.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			batch.Queue(`
				update bbl_organizations
				set kind = $2,
				    attrs = $3,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Organization.ID, a.Organization.Kind, jsonAttrs,
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

				for _, rel := range a.Organization.Rels {
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
							    rel_organization_id = $3
							where id = $1;`,
							rel.ID, rel.Kind, rel.OrganizationID,
						)
					} else {
						batch.Queue(`
							insert into bbl_organizations_rels (id, kind, organization_id, rel_organization_id)
							values ($1, $2, $3, $4);`,
							r.NewID(), rel.Kind, a.Organization.ID, rel.OrganizationID,
						)
					}
				}
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, organization_id, diff)
				values ($1, $2, $3);`,
				revID, a.Organization.ID, jsonDiff,
			)
		case *CreateProject:
			if a.Project.ID == "" {
				a.Project.ID = r.NewID()
			}

			diff := a.Project.Diff(&Project{})

			jsonAttrs, err := json.Marshal(a.Project.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			batch.Queue(`
				insert into bbl_projects (id, attrs)
				values ($1, $2);`,
				a.Project.ID, jsonAttrs,
			)
			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)
		case *UpdateProject:
			currentRec, err := getProject(ctx, tx, a.Project.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			diff := a.Project.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Project.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			batch.Queue(`
				update bbl_projects
				set attrs = $2,
				    updated_at = transaction_timestamp()
				where id = $1;`,
				a.Project.ID, jsonAttrs,
			)

			batch.Queue(`
				insert into bbl_changes (rev_id, project_id, diff)
				values ($1, $2, $3);`,
				revID, a.Project.ID, jsonDiff,
			)
		case *CreateWork:
			if a.Work.ID == "" {
				a.Work.ID = r.NewID()
			}

			diff := a.Work.Diff(&Work{})

			jsonAttrs, err := json.Marshal(a.Work.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			batch.Queue(`
				insert into bbl_works (id, kind, sub_kind, attrs)
				values ($1, $2, nullif($3, ''), $4);`,
				a.Work.ID, a.Work.Kind, a.Work.SubKind, jsonAttrs,
			)
			for _, rel := range a.Work.Rels {
				batch.Queue(`
					insert into bbl_works_rels (id, kind, work_id, rel_work_id)
					values ($1, $2, $3, $4);`,
					r.NewID(), rel.Kind, a.Work.ID, rel.WorkID,
				)
			}
			batch.Queue(`
				insert into bbl_changes (rev_id, work_id, diff)
				values ($1, $2, $3);`,
				revID, a.Work.ID, jsonDiff,
			)
		case *UpdateWork:
			currentRec, err := getWork(ctx, tx, a.Work.ID)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}

			diff := a.Work.Diff(currentRec)

			if len(diff) == 0 {
				continue
			}

			jsonAttrs, err := json.Marshal(a.Work.Attrs)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
			}
			jsonDiff, err := json.Marshal(diff)
			if err != nil {
				return fmt.Errorf("AddRev: %s", err)
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

				for _, rel := range a.Work.Rels {
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
							    rel_organization_id = $3
							where id = $1;`,
							rel.ID, rel.Kind, rel.WorkID,
						)
					} else {
						batch.Queue(`
							insert into bbl_works_rels (id, kind, work_id, rel_work_id)
							values ($1, $2, $3, $4);`,
							r.NewID(), rel.Kind, a.Work.ID, rel.WorkID,
						)
					}
				}
			}

			batch.Queue(`
				insert into bbl_changes (rev_id, work_id, diff)
				values ($1, $2, $3);`,
				revID, a.Work.ID, jsonDiff,
			)
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

func (r *Repo) NewID() string {
	return ulid.Make().UUIDString()
}

func getOrganization(ctx context.Context, conn pgxConn, id string) (*Organization, error) {
	q := `
		select id, kind, attrs, rels, created_at, updated_at
		from bbl_organizations_view
		where id = $1;`

	var rec Organization
	var rawAttrs json.RawMessage
	var rawRels json.RawMessage

	if err := conn.QueryRow(ctx, q, id).Scan(&rec.ID, &rec.Kind, &rawAttrs, &rawRels, &rec.CreatedAt, &rec.UpdatedAt); err == pgx.ErrNoRows {
		return nil, fmt.Errorf("GetOrganization: %w: %s", ErrNotFound, id)
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, fmt.Errorf("GetOrganization: unmarshal attrs: %s", err)
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, fmt.Errorf("GetOrganization: unmarshal rels: %s", err)
		}
	}

	return &rec, nil
}

func getProject(ctx context.Context, conn pgxConn, id string) (*Project, error) {
	q := `
		select id, attrs, created_at, updated_at
		from bbl_projects
		where id = $1;`

	var rec Project
	var rawAttrs json.RawMessage

	if err := conn.QueryRow(ctx, q, id).Scan(&rec.ID, &rawAttrs, &rec.CreatedAt, &rec.UpdatedAt); err == pgx.ErrNoRows {
		return nil, fmt.Errorf("Getproject: %w: %s", ErrNotFound, id)
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, fmt.Errorf("GetProject: unmarshal attrs: %s", err)
	}

	return &rec, nil
}

func getWork(ctx context.Context, conn pgxConn, id string) (*Work, error) {
	q := `
		select id, kind, coalesce(sub_kind, ''), attrs, contributors, rels, created_at, updated_at
		from bbl_works_view
		where id = $1;`

	var rec Work
	var rawAttrs json.RawMessage
	var rawContributors json.RawMessage
	var rawRels json.RawMessage

	if err := conn.QueryRow(ctx, q, id).Scan(&rec.ID, &rec.Kind, &rec.SubKind, &rawAttrs, &rawContributors, &rawRels, &rec.CreatedAt, &rec.UpdatedAt); err == pgx.ErrNoRows {
		return nil, fmt.Errorf("GetWork: %w: %s", ErrNotFound, id)
	} else if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(rawAttrs, &rec.Attrs); err != nil {
		return nil, fmt.Errorf("GetWork: unmarshal attrs: %s", err)
	}

	if rawContributors != nil {
		if err := json.Unmarshal(rawContributors, &rec.Contributors); err != nil {
			return nil, fmt.Errorf("GetWork: unmarshal contributors: %s", err)
		}
	}

	if rawRels != nil {
		if err := json.Unmarshal(rawRels, &rec.Rels); err != nil {
			return nil, fmt.Errorf("GetWork: unmarshal rels: %s", err)
		}
	}

	if err := LoadWorkProfile(&rec); err != nil {
		return nil, err
	}

	j, _ := json.Marshal(&rec)
	log.Printf("getrec: %s", j)

	return &rec, nil
}
