package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Repo) ImportPeople(ctx context.Context, source string, seq iter.Seq2[*ImportPersonInput, error]) (int, error) {
	const batchSize = 250
	var pending []*ImportPersonInput
	var total int

	flush := func() error {
		n, err := r.importPersonBatch(ctx, source, pending)
		total += n
		pending = pending[:0]
		return err
	}

	for in, err := range seq {
		if err != nil {
			return total, fmt.Errorf("ImportPeople: %w", err)
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

func (r *Repo) importPersonBatch(ctx context.Context, source string, records []*ImportPersonInput) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("importPersonBatch: %w", err)
	}
	defer tx.Rollback(ctx)

	priorities, err := fetchSourcePriorities(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("importPersonBatch: %w", err)
	}

	revID := newID()
	if _, err := tx.Exec(ctx, `
		INSERT INTO bbl_revs (id, source) VALUES ($1, $2)`,
		revID, source); err != nil {
		return 0, fmt.Errorf("importPersonBatch: %w", err)
	}

	var changedPersonIDs []ID
	var n int
	for _, in := range records {
		personID, isNew, err := r.importPersonRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("importPersonBatch: source_id=%s: %w", in.SourceID, err)
		}
		changedPersonIDs = append(changedPersonIDs, personID)
		_ = isNew
		n++
	}

	if err := rebuildPersonCache(ctx, tx, changedPersonIDs); err != nil {
		return n, fmt.Errorf("importPersonBatch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("importPersonBatch: %w", err)
	}
	return n, nil
}

func (r *Repo) importPersonRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportPersonInput, revID ID, priorities map[string]int) (ID, bool, error) {
	var personID ID
	var sourceRecordID ID
	var isNew bool
	err := tx.QueryRow(ctx, `
		SELECT person_id, id FROM bbl_person_sources
		WHERE source = $1 AND source_id = $2
		FOR UPDATE`, source, in.SourceID).Scan(&personID, &sourceRecordID)
	if errors.Is(err, pgx.ErrNoRows) {
		isNew = true
		if in.ID != nil {
			personID = *in.ID
		} else {
			personID = newID()
		}
	} else if err != nil {
		return ID{}, false, err
	}

	if isNew {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_people (id, version, status)
			VALUES ($1, 1, $2)`,
			personID, PersonStatusPublic); err != nil {
			return ID{}, false, fmt.Errorf("insert bbl_people: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_person_sources (id, person_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, personID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, false, fmt.Errorf("insert bbl_person_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, personAssertionTables, "person_source_id", sourceRecordID); err != nil {
			return ID{}, false, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_person_sources SET record = $1, ingested_at = transaction_timestamp()
			WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, false, fmt.Errorf("update bbl_person_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_people SET version = version + 1, updated_at = transaction_timestamp()
			WHERE id = $1`, personID); err != nil {
			return ID{}, false, fmt.Errorf("bump version: %w", err)
		}
	}

	// Insert scalar field assertions.
	if err := importPersonFields(ctx, tx, personID, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Insert relation assertions.
	if err := importPersonRelations(ctx, tx, personID, source, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Auto-pin all grouping keys.
	if err := autoPinAllPerson(ctx, tx, personID, priorities); err != nil {
		return ID{}, false, err
	}

	// Audit row.
	opType := OpUpdate
	if isNew {
		opType = OpCreate
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO bbl_mutations (rev_id, name, entity_type, entity_id, op_type, diff)
		VALUES ($1, $2, $3, $4, $5, '{}')`,
		revID, "ImportPerson", RecordTypePerson, personID, opType); err != nil {
		return ID{}, false, fmt.Errorf("insert bbl_mutations: %w", err)
	}

	return personID, isNew, nil
}

func importPersonFields(ctx context.Context, tx pgx.Tx, personID ID, sourceRecordID ID, in *ImportPersonInput) error {
	type sf struct {
		field string
		val   string
	}
	for _, f := range []sf{
		{"name", in.Name},
		{"given_name", in.GivenName},
		{"middle_name", in.MiddleName},
		{"family_name", in.FamilyName},
	} {
		if f.val == "" {
			continue
		}
		if err := writeCreatePersonField(ctx, tx, newID(), personID, f.field, f.val, &sourceRecordID, nil); err != nil {
			return err
		}
	}
	return nil
}

func importPersonRelations(ctx context.Context, tx pgx.Tx, personID ID, source string, sourceRecordID ID, in *ImportPersonInput) error {
	for _, id := range in.Identifiers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_person_identifiers (id, person_id, scheme, val, person_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), personID, id.Scheme, id.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_person_identifiers: %w", err)
		}
	}
	for _, a := range in.Affiliations {
		orgID, err := resolveOrganizationRef(ctx, tx, a.Ref, source)
		if err != nil {
			continue // silently skip unresolvable refs
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_person_organizations (id, person_id, organization_id, person_source_id)
			VALUES ($1, $2, $3, $4)`,
			newID(), personID, orgID, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_person_organizations: %w", err)
		}
	}
	return nil
}

// EachPerson returns an iterator over all people, ordered by id.
func (r *Repo) EachPerson(ctx context.Context) iter.Seq2[*Person, error] {
	return r.eachPerson(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, deleted_at, deleted_by_id,
		       cache
		FROM bbl_people
		ORDER BY id`)
}

// EachPersonSince returns an iterator over people updated since the given time, ordered by id.
func (r *Repo) EachPersonSince(ctx context.Context, since time.Time) iter.Seq2[*Person, error] {
	return r.eachPerson(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, deleted_at, deleted_by_id,
		       cache
		FROM bbl_people
		WHERE updated_at >= $1
		ORDER BY id`, since)
}

func (r *Repo) eachPerson(ctx context.Context, query string, args ...any) iter.Seq2[*Person, error] {
	return func(yield func(*Person, error) bool) {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("eachPerson: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			p, err := scanPersonRow(rows)
			if err != nil {
				yield(nil, fmt.Errorf("eachPerson: %w", err))
				return
			}
			if !yield(p, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("eachPerson: %w", err))
		}
	}
}

func (r *Repo) GetPerson(ctx context.Context, id ID) (*Person, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       status, deleted_at, deleted_by_id,
		       cache
		FROM bbl_people
		WHERE id = $1`, id)
	p, err := scanPerson(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetPerson: %w", err)
	}
	return p, nil
}

// scanPerson scans a single person row (including cache) from a QueryRow result.
func scanPerson(row pgx.Row) (*Person, error) {
	var p Person
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		p.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		p.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		p.DeletedByID = &id
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if err := parsePersonCache(&p, cache); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanPersonRow(row pgx.CollectableRow) (*Person, error) {
	var p Person
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		p.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		p.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		p.DeletedByID = &id
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if err := parsePersonCache(&p, cache); err != nil {
		return nil, err
	}
	return &p, nil
}

func parsePersonCache(p *Person, cache []byte) error {
	if len(cache) == 0 || string(cache) == "{}" {
		return nil
	}
	var d struct {
		Name          string               `json:"name,omitempty"`
		GivenName     string               `json:"given_name,omitempty"`
		MiddleName    string               `json:"middle_name,omitempty"`
		FamilyName    string               `json:"family_name,omitempty"`
		Identifiers   []Identifier         `json:"identifiers,omitempty"`
		Organizations []PersonOrganization `json:"organizations,omitempty"`
	}
	if err := json.Unmarshal(cache, &d); err != nil {
		return fmt.Errorf("parsePersonCache: %w", err)
	}
	p.Name = d.Name
	p.GivenName = d.GivenName
	p.MiddleName = d.MiddleName
	p.FamilyName = d.FamilyName
	p.Identifiers = d.Identifiers
	p.Organizations = d.Organizations
	return nil
}
