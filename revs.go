package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RevEffect describes a record affected by a revision.
type RevEffect struct {
	RecordType string // "work", "person", "project", "organization"
	Record     any    // *Work, *Person, *Project, *Organization
}

// AddRev executes a batch of mutations atomically.
// Each mutation must be a pointer to a type implementing the unexported mutation
// interface (e.g. *CreateWork, *DeletePerson, *CreateWorkTitle).
//
// Returns (true, effects, nil) when a rev was written, (false, nil, nil) when
// every mutation was a noop.
//
// Execution order:
//  1. Type-assert each arg to mutation interface
//  2. Collect needs → batch prefetch with FOR UPDATE
//  3. Apply all mutations (pure, no DB)
//  4. If all nil → rollback, return (false, nil, nil)
//  5. Insert bbl_revs row
//  6. Write each non-noop mutation + audit rows
//  7. Bump version for entities affected only by field mutations
//  8. Run auto-pin for affected grouping keys
//  9. Rebuild cache for affected entities
//  10. Return deduplicated RevEffects
func (r *Repo) Mutate(ctx context.Context, userID *ID, mutations ...any) (bool, []RevEffect, error) {
	// 1. Type-assert to mutation interface.
	muts := make([]mutation, len(mutations))
	for i, arg := range mutations {
		m, ok := arg.(mutation)
		if !ok {
			return false, nil, fmt.Errorf("Mutate: argument %d (%T) does not implement mutation", i, arg)
		}
		muts[i] = m
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Collect needs and batch prefetch.
	var needs mutationNeeds
	for _, m := range muts {
		n := m.needs()
		needs.organizationIDs = append(needs.organizationIDs, n.organizationIDs...)
		needs.personIDs = append(needs.personIDs, n.personIDs...)
		needs.projectIDs = append(needs.projectIDs, n.projectIDs...)
		needs.workIDs = append(needs.workIDs, n.workIDs...)
	}

	state, err := fetchMutationState(ctx, tx, needs)
	if err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}

	// 3. Apply all mutations (pure — no DB access).
	effects := make([]*mutationEffect, len(muts))
	for i, m := range muts {
		eff, err := m.apply(state, userID)
		if err != nil {
			return false, nil, fmt.Errorf("Mutate: %s: %w", m.mutationName(), err)
		}
		effects[i] = eff // nil = noop
	}

	// 4. If all nil, nothing to write.
	allNoop := true
	for _, eff := range effects {
		if eff != nil {
			allNoop = false
			break
		}
	}
	if allNoop {
		return false, nil, nil
	}

	// Set actor IDs on non-noop entity lifecycle records.
	if userID != nil {
		for _, eff := range effects {
			if eff == nil || eff.record == nil {
				continue
			}
			setRecordActorIDs(eff, userID)
		}
	}

	// Validate non-noop entity lifecycle records.
	for i, eff := range effects {
		if eff == nil || eff.record == nil {
			continue
		}
		if err := r.validateRecord(eff); err != nil {
			return false, nil, fmt.Errorf("Mutate: %s: %w", muts[i].mutationName(), err)
		}
	}

	// 5. Insert bbl_revs row.
	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (user_id) VALUES ($1) RETURNING id`,
		userID).Scan(&revID); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}

	// 6. Write each non-noop mutation.
	// Track which entities were touched by lifecycle mutations (record != nil)
	// vs field-only mutations (record == nil).
	var changedWorkIDs, changedPersonIDs, changedProjectIDs, changedOrganizationIDs []ID
	lifecycleEntities := make(map[string]map[ID]bool) // recordType → recordID → true
	seen := make(map[string]map[ID]bool)              // all affected entities
	hasAutoPin := false
	for i, eff := range effects {
		if eff == nil {
			continue
		}
		if err := muts[i].write(ctx, tx, revID); err != nil {
			return false, nil, fmt.Errorf("Mutate: %w", err)
		}

		// Track affected entities.
		if seen[eff.recordType] == nil {
			seen[eff.recordType] = make(map[ID]bool)
		}
		seen[eff.recordType][eff.recordID] = true

		if eff.record != nil {
			// Lifecycle mutation — entity row already written with version bump.
			if lifecycleEntities[eff.recordType] == nil {
				lifecycleEntities[eff.recordType] = make(map[ID]bool)
			}
			lifecycleEntities[eff.recordType][eff.recordID] = true
		}

		switch eff.recordType {
		case RecordTypeWork:
			changedWorkIDs = append(changedWorkIDs, eff.recordID)
		case RecordTypePerson:
			changedPersonIDs = append(changedPersonIDs, eff.recordID)
		case RecordTypeProject:
			changedProjectIDs = append(changedProjectIDs, eff.recordID)
		case RecordTypeOrganization:
			changedOrganizationIDs = append(changedOrganizationIDs, eff.recordID)
		}
		if eff.autoPin != nil {
			hasAutoPin = true
		}
	}

	// 7. Bump version + updated_at for entities affected only by field mutations.
	for rt, ids := range seen {
		for id := range ids {
			if lifecycleEntities[rt][id] {
				continue // lifecycle mutation already bumped version
			}
			var q string
			switch rt {
			case RecordTypeWork:
				q = `UPDATE bbl_works SET version = version + 1, updated_at = transaction_timestamp(), updated_by_id = $2 WHERE id = $1`
			case RecordTypePerson:
				q = `UPDATE bbl_people SET version = version + 1, updated_at = transaction_timestamp(), updated_by_id = $2 WHERE id = $1`
			case RecordTypeProject:
				q = `UPDATE bbl_projects SET version = version + 1, updated_at = transaction_timestamp(), updated_by_id = $2 WHERE id = $1`
			case RecordTypeOrganization:
				q = `UPDATE bbl_organizations SET version = version + 1, updated_at = transaction_timestamp(), updated_by_id = $2 WHERE id = $1`
			}
			if q != "" {
				if _, err := tx.Exec(ctx, q, id, userID); err != nil {
					return false, nil, fmt.Errorf("Mutate: bump version %s %s: %w", rt, id, err)
				}
			}
		}
	}

	// 8. Run auto-pin for affected grouping keys.
	if hasAutoPin {
		priorities, err := fetchSourcePriorities(ctx, tx)
		if err != nil {
			return false, nil, fmt.Errorf("Mutate: %w", err)
		}
		for _, eff := range effects {
			if eff == nil || eff.autoPin == nil {
				continue
			}
			if err := eff.autoPin(ctx, tx, priorities); err != nil {
				return false, nil, fmt.Errorf("Mutate: autoPin: %w", err)
			}
		}
	}

	// 9. Rebuild caches for affected entities.
	if err := rebuildWorkCache(ctx, tx, changedWorkIDs); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}
	if err := rebuildPersonCache(ctx, tx, changedPersonIDs); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}
	if err := rebuildProjectCache(ctx, tx, changedProjectIDs); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}
	if err := rebuildOrganizationCache(ctx, tx, changedOrganizationIDs); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, nil, fmt.Errorf("Mutate: %w", err)
	}

	// 10. Build deduplicated RevEffects.
	// For lifecycle mutations, use the in-memory record. For field-only
	// mutations, the record is nil — callers that need the full entity
	// should re-read from the repo.
	var revEffects []RevEffect
	for rt, ids := range seen {
		for id := range ids {
			var rec any
			// Find the record from lifecycle effects if available.
			for _, eff := range effects {
				if eff != nil && eff.recordType == rt && eff.recordID == id && eff.record != nil {
					rec = eff.record
					break
				}
			}
			revEffects = append(revEffects, RevEffect{RecordType: rt, Record: rec})
		}
	}
	return true, revEffects, nil
}

// fetchSourcePriorities reads all source priorities from bbl_sources.
func fetchSourcePriorities(ctx context.Context, tx pgx.Tx) (map[string]int, error) {
	rows, err := tx.Query(ctx, `SELECT id, priority FROM bbl_sources`)
	if err != nil {
		return nil, fmt.Errorf("fetchSourcePriorities: %w", err)
	}
	defer rows.Close()
	priorities := make(map[string]int)
	for rows.Next() {
		var id string
		var p int
		if err := rows.Scan(&id, &p); err != nil {
			return nil, fmt.Errorf("fetchSourcePriorities: %w", err)
		}
		priorities[id] = p
	}
	return priorities, rows.Err()
}

func fetchMutationState(ctx context.Context, tx pgx.Tx, needs mutationNeeds) (mutationState, error) {
	state := mutationState{}
	if len(needs.organizationIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT id, version, created_at, updated_at, created_by_id, updated_by_id,
			       kind, status, start_date, end_date,
			       deleted_at, deleted_by_id
			FROM bbl_organizations
			WHERE id = ANY($1)
			FOR UPDATE`, dedup(needs.organizationIDs))
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		orgs, err := pgx.CollectRows(rows, scanOrganizationRow)
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		state.organizations = make(map[ID]*Organization, len(orgs))
		for _, o := range orgs {
			state.organizations[o.ID] = o
		}
	}
	if len(needs.personIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT id, version, created_at, updated_at, created_by_id, updated_by_id,
			       status, deleted_at, deleted_by_id
			FROM bbl_people
			WHERE id = ANY($1)
			FOR UPDATE`, dedup(needs.personIDs))
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		people, err := pgx.CollectRows(rows, scanPersonMutationRow)
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		state.people = make(map[ID]*Person, len(people))
		for _, p := range people {
			state.people[p.ID] = p
		}
	}
	if len(needs.workIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT id, version, created_at, updated_at, created_by_id, updated_by_id,
			       kind, status, review_status, delete_kind,
			       deleted_at, deleted_by_id
			FROM bbl_works
			WHERE id = ANY($1)
			FOR UPDATE`, dedup(needs.workIDs))
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		works, err := pgx.CollectRows(rows, scanWorkMutationRow)
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		state.works = make(map[ID]*Work, len(works))
		for _, w := range works {
			state.works[w.ID] = w
		}
	}
	if len(needs.projectIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT id, version, created_at, updated_at, created_by_id, updated_by_id,
			       status, start_date, end_date,
			       deleted_at, deleted_by_id
			FROM bbl_projects
			WHERE id = ANY($1)
			FOR UPDATE`, dedup(needs.projectIDs))
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		projects, err := pgx.CollectRows(rows, scanProjectMutationRow)
		if err != nil {
			return state, fmt.Errorf("fetchMutationState: %w", err)
		}
		state.projects = make(map[ID]*Project, len(projects))
		for _, p := range projects {
			state.projects[p.ID] = p
		}
	}
	return state, nil
}

// scanWorkMutationRow scans a work row for mutation state (no cache column).
func scanWorkMutationRow(row pgx.CollectableRow) (*Work, error) {
	var w Work
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var reviewStatus, deleteKind pgtype.Text
	var deletedAt pgtype.Timestamptz
	err := row.Scan(
		&w.ID, &w.Version, &w.CreatedAt, &w.UpdatedAt,
		&createdByID, &updatedByID,
		&w.Kind, &w.Status, &reviewStatus, &deleteKind,
		&deletedAt, &deletedByID,
	)
	if err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		w.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		w.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		w.DeletedByID = &id
	}
	if reviewStatus.Valid {
		w.ReviewStatus = reviewStatus.String
	}
	if deleteKind.Valid {
		w.DeleteKind = deleteKind.String
	}
	if deletedAt.Valid {
		w.DeletedAt = &deletedAt.Time
	}
	return &w, nil
}

// scanPersonMutationRow scans a person row for mutation state.
func scanPersonMutationRow(row pgx.CollectableRow) (*Person, error) {
	var p Person
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var deletedAt pgtype.Timestamptz
	err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &deletedAt, &deletedByID,
	)
	if err != nil {
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
	return &p, nil
}

// scanProjectMutationRow scans a project row for mutation state.
func scanProjectMutationRow(row pgx.CollectableRow) (*Project, error) {
	var p Project
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var startDate, endDate, deletedAt pgtype.Timestamptz
	err := row.Scan(
		&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt,
		&createdByID, &updatedByID,
		&p.Status, &startDate, &endDate,
		&deletedAt, &deletedByID,
	)
	if err != nil {
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
	if startDate.Valid {
		p.StartDate = &startDate.Time
	}
	if endDate.Valid {
		p.EndDate = &endDate.Time
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	return &p, nil
}

// setRecordActorIDs sets created_by_id, updated_by_id, and deleted_by_id on the
// record based on the record's current state and the acting user.
func setRecordActorIDs(eff *mutationEffect, userID *ID) {
	switch rec := eff.record.(type) {
	case *Work:
		if rec.CreatedByID == nil {
			rec.CreatedByID = userID
		}
		rec.UpdatedByID = userID
		if rec.Status == WorkStatusDeleted {
			rec.DeletedByID = userID
		}
	case *Person:
		if rec.CreatedByID == nil {
			rec.CreatedByID = userID
		}
		rec.UpdatedByID = userID
		if rec.Status == PersonStatusDeleted {
			rec.DeletedByID = userID
		}
	case *Project:
		if rec.CreatedByID == nil {
			rec.CreatedByID = userID
		}
		rec.UpdatedByID = userID
		if rec.Status == ProjectStatusDeleted {
			rec.DeletedByID = userID
		}
	case *Organization:
		if rec.CreatedByID == nil {
			rec.CreatedByID = userID
		}
		rec.UpdatedByID = userID
		if rec.Status == OrganizationStatusDeleted {
			rec.DeletedByID = userID
		}
	}
}

// validateRecord runs entity validation on a mutation result.
func (r *Repo) validateRecord(eff *mutationEffect) error {
	switch rec := eff.record.(type) {
	case *Work:
		if r.WorkProfiles != nil {
			if errs := ValidateWork(rec, r.WorkProfiles); errs != nil {
				return errs.ToError()
			}
		}
	case *Person:
		if errs := ValidatePerson(rec); errs != nil {
			return errs.ToError()
		}
	case *Project:
		if errs := ValidateProject(rec); errs != nil {
			return errs.ToError()
		}
	case *Organization:
		if errs := ValidateOrganization(rec); errs != nil {
			return errs.ToError()
		}
	}
	return nil
}

// nilIfEmpty returns nil for empty strings (for nullable text columns).
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
