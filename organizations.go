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

func (r *Repo) GetOrganization(ctx context.Context, id ID) (*Organization, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_organizations
		WHERE id = $1`, id)
	o, err := scanOrganization(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetOrganization: %w", err)
	}
	return o, nil
}

func (r *Repo) ImportOrganizations(ctx context.Context, source string, seq iter.Seq2[*ImportOrganizationInput, error]) (int, error) {
	const batchSize = 250
	var pending []*ImportOrganizationInput
	var total int

	flush := func() error {
		n, err := r.importOrganizationBatch(ctx, source, pending)
		total += n
		pending = pending[:0]
		return err
	}

	for in, err := range seq {
		if err != nil {
			return total, fmt.Errorf("ImportOrganizations: %w", err)
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

func (r *Repo) importOrganizationBatch(ctx context.Context, source string, records []*ImportOrganizationInput) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("importOrganizationBatch: %w", err)
	}
	defer tx.Rollback(ctx)

	priorities, err := fetchSourcePriorities(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("importOrganizationBatch: %w", err)
	}

	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (source) VALUES ($1) RETURNING id`,
		source).Scan(&revID); err != nil {
		return 0, fmt.Errorf("importOrganizationBatch: %w", err)
	}

	var changedOrgIDs []ID
	var n int
	for _, in := range records {
		orgID, err := r.importOrganizationRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("importOrganizationBatch: source_id=%s: %w", in.SourceID, err)
		}
		changedOrgIDs = append(changedOrgIDs, orgID)
		n++
	}

	if err := rebuildOrganizationCache(ctx, tx, changedOrgIDs); err != nil {
		return n, fmt.Errorf("importOrganizationBatch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("importOrganizationBatch: %w", err)
	}
	return n, nil
}

func (r *Repo) importOrganizationRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportOrganizationInput, revID int64, priorities map[string]int) (ID, error) {
	var orgID ID
	var sourceRecordID ID
	var isNew bool
	err := tx.QueryRow(ctx, `
		SELECT organization_id, id FROM bbl_organization_sources
		WHERE source = $1 AND source_id = $2
		FOR UPDATE`, source, in.SourceID).Scan(&orgID, &sourceRecordID)
	if errors.Is(err, pgx.ErrNoRows) {
		isNew = true
		if in.ID != nil {
			orgID = *in.ID
		} else {
			orgID = newID()
		}
	} else if err != nil {
		return ID{}, err
	}

	if isNew {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_organizations (id, version, kind, status, start_date, end_date)
			VALUES ($1, 1, $2, $3, $4, $5)`,
			orgID, in.Kind, OrganizationStatusPublic, in.StartDate, in.EndDate); err != nil {
			return ID{}, fmt.Errorf("insert bbl_organizations: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_organization_sources (id, organization_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, orgID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, fmt.Errorf("insert bbl_organization_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, "bbl_organization_assertions", "organization_source_id", sourceRecordID); err != nil {
			return ID{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_organization_sources SET record = $1, ingested_at = transaction_timestamp()
			WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, fmt.Errorf("update bbl_organization_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_organizations SET version = version + 1, updated_at = transaction_timestamp(),
			       kind = $2, start_date = $3, end_date = $4
			WHERE id = $1`, orgID, in.Kind, in.StartDate, in.EndDate); err != nil {
			return ID{}, fmt.Errorf("bump version: %w", err)
		}
	}

	// Build assertion rows, validate, write via shared pipeline.
	rows, err := organizationImportAssertions(ctx, tx, source, orgID, sourceRecordID, in)
	if err != nil {
		return ID{}, err
	}
	if defs := r.Profiles.FieldDefs(RecordTypeOrganization, in.Kind); defs != nil {
		if errs := validateRecord(OrganizationStatusPublic, assertionRowFields(rows), defs); errs != nil {
			return ID{}, errs.ToError()
		}
	}
	if err := writeAssertionRows(ctx, tx, &pgx.Batch{}, 0, revID, rows); err != nil {
		return ID{}, err
	}

	// Auto-pin all grouping keys.
	if err := autoPinRecord(ctx, tx, RecordTypeOrganization, orgID, priorities); err != nil {
		return ID{}, err
	}

	return orgID, nil
}

// organizationImportAssertions resolves refs and converts an ImportOrganizationInput into assertion rows.
func organizationImportAssertions(ctx context.Context, tx pgx.Tx, source string, orgID ID, sourceRecordID ID, in *ImportOrganizationInput) ([]assertionRow, error) {
	var rows []assertionRow
	src := &sourceRecordID

	if len(in.Identifiers) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeOrganization, recordID: orgID,
			field: "identifiers", val: in.Identifiers, sourceRecordID: src,
		})
	}
	if len(in.Names) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeOrganization, recordID: orgID,
			field: "names", val: in.Names, sourceRecordID: src,
		})
	}
	if len(in.Rels) > 0 {
		orgRels := make([]OrganizationRel, 0, len(in.Rels))
		for _, rel := range in.Rels {
			relOrg, err := resolveOrganizationRef(ctx, tx, rel.Ref, source)
			if err != nil {
				return nil, fmt.Errorf("organizationImportAssertions: resolve organization ref: %w", err)
			}
			orgRels = append(orgRels, OrganizationRel{
				RelOrganizationID: relOrg.ID, Kind: rel.Kind,
				StartDate: rel.StartDate, EndDate: rel.EndDate,
			})
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeOrganization, recordID: orgID,
			field: "rels", val: orgRels, sourceRecordID: src,
		})
	}
	return rows, nil
}

// EachOrganization returns an iterator over all organizations, ordered by id.
func (r *Repo) EachOrganization(ctx context.Context) iter.Seq2[*Organization, error] {
	return r.eachOrganization(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_organizations
		ORDER BY id`)
}

// EachOrganizationSince returns an iterator over organizations updated since the given time, ordered by id.
func (r *Repo) EachOrganizationSince(ctx context.Context, since time.Time) iter.Seq2[*Organization, error] {
	return r.eachOrganization(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, start_date, end_date,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_organizations
		WHERE updated_at >= $1
		ORDER BY id`, since)
}

func (r *Repo) eachOrganization(ctx context.Context, query string, args ...any) iter.Seq2[*Organization, error] {
	return func(yield func(*Organization, error) bool) {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("eachOrganization: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			o, err := scanOrganization(rows)
			if err != nil {
				yield(nil, fmt.Errorf("eachOrganization: %w", err))
				return
			}
			if !yield(o, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("eachOrganization: %w", err))
		}
	}
}

// scanOrganization scans a single organization row (including cache) from a QueryRow result.
func scanOrganization(row pgx.Row) (*Organization, error) {
	var o Organization
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&o.ID, &o.Version, &o.CreatedAt, &o.UpdatedAt,
		&createdByID, &updatedByID,
		&o.Kind, &o.Status, &o.StartDate, &o.EndDate,
		&deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		o.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		o.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		o.DeletedByID = &id
	}
	if deletedAt.Valid {
		o.DeletedAt = &deletedAt.Time
	}
	if err := parseOrganizationCache(&o, cache); err != nil {
		return nil, err
	}
	return &o, nil
}

func parseOrganizationCache(o *Organization, cache []byte) error {
	if len(cache) == 0 || string(cache) == "{}" {
		return nil
	}
	var d struct {
		Identifiers []Identifier          `json:"identifiers,omitempty"`
		Names       []Text                `json:"names,omitempty"`
		Rels        []OrganizationRel     `json:"rels,omitempty"`
	}
	if err := json.Unmarshal(cache, &d); err != nil {
		return fmt.Errorf("parseOrganizationCache: %w", err)
	}
	o.Identifiers = d.Identifiers
	o.Names = d.Names
	o.Rels = d.Rels
	return nil
}
