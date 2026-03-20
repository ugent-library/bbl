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
	// Collect all records first — two-phase import requires all orgs to exist before rels.
	var records []*ImportOrganizationInput
	for in, err := range seq {
		if err != nil {
			return 0, fmt.Errorf("ImportOrganizations: %w", err)
		}
		records = append(records, in)
	}
	if len(records) == 0 {
		return 0, nil
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("ImportOrganizations: %w", err)
	}
	defer tx.Rollback(ctx)

	priorities, err := fetchSourcePriorities(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("ImportOrganizations: %w", err)
	}

	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (source) VALUES ($1) RETURNING id`,
		source).Scan(&revID); err != nil {
		return 0, fmt.Errorf("ImportOrganizations: %w", err)
	}

	// Phase 1: Create/update orgs with identifiers and titles (no rels).
	type orgInfo struct {
		orgID, srcRecID ID
	}
	orgInfos := make(map[string]orgInfo, len(records)) // source_id → orgInfo
	var changedOrgIDs []ID
	var n int
	for _, in := range records {
		orgID, srcRecID, isNew, err := r.importOrganizationRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("ImportOrganizations: source_id=%s: %w", in.SourceID, err)
		}
		orgInfos[in.SourceID] = orgInfo{orgID: orgID, srcRecID: srcRecID}
		changedOrgIDs = append(changedOrgIDs, orgID)
		_ = isNew
		n++
	}

	// Phase 2: Insert rels (all orgs exist now, refs can be resolved).
	for _, in := range records {
		info := orgInfos[in.SourceID]
		orgID := info.orgID
		srcRecID := info.srcRecID
		if len(in.Rels) > 0 {
			for _, rel := range in.Rels {
				relOrg, err := resolveOrganizationRef(ctx, tx, rel.Ref, source)
				if err != nil {
					continue
				}
				val := struct {
					Kind string `json:"kind"`
				}{rel.Kind}
				aID, err := writeOrganizationAssertion(ctx, tx, revID, orgID, "rels", val, false, &srcRecID, nil, nil)
				if err != nil {
					return n, err
				}
				if err := writeOrganizationRel(ctx, tx, aID, relOrg.ID, rel.Kind); err != nil {
					return n, err
				}
			}
		}
		// Re-run auto-pin for rels (phase 1 already pinned identifiers+titles).
		if len(in.Rels) > 0 {
			if err := autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", orgID, "rels", "organization_source_id", "bbl_organization_sources", priorities); err != nil {
				return n, err
			}
		}
	}

	if err := rebuildOrganizationCache(ctx, tx, changedOrgIDs); err != nil {
		return n, fmt.Errorf("ImportOrganizations: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("ImportOrganizations: %w", err)
	}
	return n, nil
}

func (r *Repo) importOrganizationRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportOrganizationInput, revID int64, priorities map[string]int) (ID, ID, bool, error) {
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
		return ID{}, ID{}, false, err
	}

	if isNew {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_organizations (id, version, kind, status, start_date, end_date)
			VALUES ($1, 1, $2, $3, $4, $5)`,
			orgID, in.Kind, OrganizationStatusPublic, in.StartDate, in.EndDate); err != nil {
			return ID{}, ID{}, false, fmt.Errorf("insert bbl_organizations: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_organization_sources (id, organization_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, orgID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, ID{}, false, fmt.Errorf("insert bbl_organization_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, "bbl_organization_assertions", "organization_source_id", sourceRecordID); err != nil {
			return ID{}, ID{}, false, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_organization_sources SET record = $1, ingested_at = transaction_timestamp()
			WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, ID{}, false, fmt.Errorf("update bbl_organization_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_organizations SET version = version + 1, updated_at = transaction_timestamp(),
			       kind = $2, start_date = $3, end_date = $4
			WHERE id = $1`, orgID, in.Kind, in.StartDate, in.EndDate); err != nil {
			return ID{}, ID{}, false, fmt.Errorf("bump version: %w", err)
		}
	}

	// Insert identifiers.
	if len(in.Identifiers) > 0 {
		for _, id := range in.Identifiers {
			if _, err := writeOrganizationAssertion(ctx, tx, revID, orgID, "identifiers", id, false, &sourceRecordID, nil, nil); err != nil {
				return ID{}, ID{}, false, err
			}
		}
	}

	// Insert names.
	if len(in.Names) > 0 {
		for _, t := range in.Names {
			if _, err := writeOrganizationAssertion(ctx, tx, revID, orgID, "names", t, false, &sourceRecordID, nil, nil); err != nil {
				return ID{}, ID{}, false, err
			}
		}
	}

	// Auto-pin identifiers and titles (rels handled in phase 2).
	if err := autoPinAllOrganization(ctx, tx, orgID, priorities); err != nil {
		return ID{}, ID{}, false, err
	}

	return orgID, sourceRecordID, isNew, nil
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
			o, err := scanOrganizationRow(rows)
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

func scanOrganizationRow(row pgx.CollectableRow) (*Organization, error) {
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
