package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// RevEffect describes a record affected by a revision.
type RevEffect struct {
	RecordType string
	RecordID   ID
	Version    int
}

// Update executes a batch of updaters atomically.
//
// Flow:
//  1. Parse updaters
//  2. Collect needs → fetch state (lock rows + pinned field state → recordState)
//  3. Apply all updaters (mutate recordState)
//  4. If all noop → return (false, nil, nil)
//  5. Validate (rs.fields + profile defs)
//  6. Insert bbl_revs row
//  7. Write all (field batch + lifecycle writes)
//  8. Bump version + updated_at + updated_by_id for all existing affected records
//  9. Auto-pin for affected grouping keys
//  10. Rebuild cache for affected entities
//  11. Commit
//  12. Return (true, []RevEffect, nil)
func (r *Repo) Update(ctx context.Context, user *User, updates ...any) (bool, []RevEffect, error) {
	// 1. Parse updaters.
	muts := make([]updater, len(updates))
	for i, arg := range updates {
		m, ok := arg.(updater)
		if !ok {
			return false, nil, fmt.Errorf("Update: argument %d (%T) does not implement updater", i, arg)
		}
		muts[i] = m
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}
	defer tx.Rollback(ctx)

	// 2. Fetch state: lock rows + pinned field state → recordState.
	var needs updateNeeds
	for _, m := range muts {
		n := m.needs()
		needs.organizationIDs = append(needs.organizationIDs, n.organizationIDs...)
		needs.personIDs = append(needs.personIDs, n.personIDs...)
		needs.projectIDs = append(needs.projectIDs, n.projectIDs...)
		needs.workIDs = append(needs.workIDs, n.workIDs...)
	}

	state, err := fetchState(ctx, tx, needs, muts)
	if err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}

	// 3. Apply all updaters.
	effects := make([]*updateEffect, len(muts))
	for i, m := range muts {
		eff, err := m.apply(state, user)
		if err != nil {
			return false, nil, fmt.Errorf("Update: %s: %w", m.name(), err)
		}
		effects[i] = eff
	}

	// 4. If all noop, nothing to write.
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

	// 5. Validate.
	if r.Profiles != nil {
		validated := make(map[ID]bool)
		for _, eff := range effects {
			if eff == nil || validated[eff.recordID] {
				continue
			}
			validated[eff.recordID] = true

			rs := state.records[eff.recordID]
			if rs == nil {
				continue
			}

			defs := r.Profiles.FieldDefs(eff.recordType, rs.kind)
			if defs == nil {
				continue
			}

			if errs := validateRecord(rs.status, rs.fields, defs); errs != nil {
				return false, nil, errs.ToError()
			}
		}
	}

	// 6. Insert bbl_revs row.
	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (user_id) VALUES ($1) RETURNING id`,
		&user.ID).Scan(&revID); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}

	// 7. Write all.
	// Field ops go through the assertion batch pipeline.
	if err := executeFieldWrites(ctx, tx, revID, user, muts, effects); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}

	// Lifecycle ops + version bumps go through a single batch.
	affected := make(map[ID]string) // id → recordType
	for _, eff := range effects {
		if eff != nil {
			affected[eff.recordID] = eff.recordType
		}
	}

	var changedWorkIDs, changedPersonIDs, changedProjectIDs, changedOrganizationIDs []ID
	{
		batch := &pgx.Batch{}

		// Queue lifecycle writes.
		for i, eff := range effects {
			if eff == nil {
				continue
			}
			switch muts[i].(type) {
			case *Set, *Hide, *Unset:
				continue
			}
			if sql, args := muts[i].write(revID, user); sql != "" {
				batch.Queue(sql, args...)
			}
		}

		// Queue version bumps for existing records.
		for id, rt := range affected {
			switch rt {
			case RecordTypeWork:
				changedWorkIDs = append(changedWorkIDs, id)
			case RecordTypePerson:
				changedPersonIDs = append(changedPersonIDs, id)
			case RecordTypeProject:
				changedProjectIDs = append(changedProjectIDs, id)
			case RecordTypeOrganization:
				changedOrganizationIDs = append(changedOrganizationIDs, id)
			}

			rs := state.records[id]
			if rs == nil || rs.version == 0 {
				continue // new entity — Create already inserted with version=1
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
				batch.Queue(q, id, &user.ID)
			}
		}

		if batch.Len() > 0 {
			results := tx.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				if _, err := results.Exec(); err != nil {
					results.Close()
					return false, nil, fmt.Errorf("Update: write batch: %w", err)
				}
			}
			if err := results.Close(); err != nil {
				return false, nil, fmt.Errorf("Update: close write batch: %w", err)
			}
		}
	}

	// 9. Auto-pin.
	hasAutoPin := false
	for _, eff := range effects {
		if eff != nil && eff.autoPin != nil {
			hasAutoPin = true
			break
		}
	}
	if hasAutoPin {
		priorities, err := fetchSourcePriorities(ctx, tx)
		if err != nil {
			return false, nil, fmt.Errorf("Update: %w", err)
		}
		for _, eff := range effects {
			if eff == nil || eff.autoPin == nil {
				continue
			}
			if err := eff.autoPin(ctx, tx, priorities); err != nil {
				return false, nil, fmt.Errorf("Update: autoPin: %w", err)
			}
		}
	}

	// 10. Rebuild caches.
	if err := rebuildWorkCache(ctx, tx, changedWorkIDs); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}
	if err := rebuildPersonCache(ctx, tx, changedPersonIDs); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}
	if err := rebuildProjectCache(ctx, tx, changedProjectIDs); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}
	if err := rebuildOrganizationCache(ctx, tx, changedOrganizationIDs); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}

	// 11. Commit.
	if err := tx.Commit(ctx); err != nil {
		return false, nil, fmt.Errorf("Update: %w", err)
	}

	// 12. Return effects.
	revEffects := make([]RevEffect, 0, len(affected))
	for id, rt := range affected {
		version := 1 // new entities
		if rs := state.records[id]; rs != nil && rs.version > 0 {
			version = rs.version + 1
		}
		revEffects = append(revEffects, RevEffect{
			RecordType: rt,
			RecordID:   id,
			Version:    version,
		})
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

// fetchState locks entity rows and fetches pinned field state in one pass.
// Builds map[ID]*recordState with status, kind, version, and field values.
func fetchState(ctx context.Context, tx pgx.Tx, needs updateNeeds, muts []updater) (updateState, error) {
	state := updateState{
		records: make(map[ID]*recordState),
	}

	// Phase 1: lock rows and build recordState stubs.
	type lockQuery struct {
		rt  string
		ids []ID
		sql string
	}
	var lockQueries []lockQuery
	if len(needs.workIDs) > 0 {
		lockQueries = append(lockQueries, lockQuery{
			rt: RecordTypeWork, ids: dedupIDs(needs.workIDs),
			sql: `SELECT id, version, kind, status FROM bbl_works WHERE id = ANY($1) FOR UPDATE`,
		})
	}
	if len(needs.personIDs) > 0 {
		lockQueries = append(lockQueries, lockQuery{
			rt: RecordTypePerson, ids: dedupIDs(needs.personIDs),
			sql: `SELECT id, version, '', status FROM bbl_people WHERE id = ANY($1) FOR UPDATE`,
		})
	}
	if len(needs.projectIDs) > 0 {
		lockQueries = append(lockQueries, lockQuery{
			rt: RecordTypeProject, ids: dedupIDs(needs.projectIDs),
			sql: `SELECT id, version, '', status FROM bbl_projects WHERE id = ANY($1) FOR UPDATE`,
		})
	}
	if len(needs.organizationIDs) > 0 {
		lockQueries = append(lockQueries, lockQuery{
			rt: RecordTypeOrganization, ids: dedupIDs(needs.organizationIDs),
			sql: `SELECT id, version, kind, status FROM bbl_organizations WHERE id = ANY($1) FOR UPDATE`,
		})
	}

	if len(lockQueries) > 0 {
		batch := &pgx.Batch{}
		for _, lq := range lockQueries {
			batch.Queue(lq.sql, lq.ids)
		}
		results := tx.SendBatch(ctx, batch)
		for _, lq := range lockQueries {
			rows, err := results.Query()
			if err != nil {
				results.Close()
				return state, fmt.Errorf("fetchState: lock %s: %w", lq.rt, err)
			}
			for rows.Next() {
				var id ID
				var version int
				var kind, status string
				if err := rows.Scan(&id, &version, &kind, &status); err != nil {
					rows.Close()
					results.Close()
					return state, fmt.Errorf("fetchState: scan %s: %w", lq.rt, err)
				}
				state.records[id] = &recordState{
					recordType: lq.rt,
					id:         id,
					version:    version,
					status:     status,
					kind:       kind,
					fields:     make(map[string]any),
					assertions: make(map[string]*fieldState),
				}
			}
			rows.Close()
		}
		if err := results.Close(); err != nil {
			return state, fmt.Errorf("fetchState: close lock: %w", err)
		}
	}

	// Phase 2: fetch pinned assertion state for fields being updated.
	type entityKey struct {
		rt string
		id ID
	}
	grouped := make(map[entityKey][]string)
	for _, m := range muts {
		switch u := m.(type) {
		case *Set:
			ek := entityKey{u.RecordType, u.RecordID}
			grouped[ek] = append(grouped[ek], u.Field)
		case *Hide:
			ek := entityKey{u.RecordType, u.RecordID}
			grouped[ek] = append(grouped[ek], u.Field)
		case *Unset:
			ek := entityKey{u.RecordType, u.RecordID}
			grouped[ek] = append(grouped[ek], u.Field)
		}
	}

	if len(grouped) > 0 {
		type rrInfo struct {
			rr     *relation
			offset int // start index in the extra scan slice
		}
		type queryInfo struct {
			ek      entityKey
			fields  []string
			joins   string
			sel     string              // full SELECT clause
			rrByFT  map[string]*rrInfo  // fieldType name → rrInfo (deduped by rr pointer)
			rrOrder []*rrInfo           // insertion order for stable scan dest building
		}

		batch := &pgx.Batch{}
		var queries []queryInfo

		for ek, fields := range grouped {
			qi := queryInfo{
				ek:     ek,
				fields: dedupStrings(fields),
				rrByFT: make(map[string]*rrInfo),
			}

			// Collect unique relations for the requested fields.
			extraOffset := 0
			seen := make(map[*relation]*rrInfo)
			var extraCols []string
			for _, f := range qi.fields {
				ft, err := resolveFieldType(ek.rt, f)
				if err != nil || ft.relation == nil {
					continue
				}
				rr := ft.relation
				ri, ok := seen[rr]
				if !ok {
					ri = &rrInfo{rr: rr, offset: extraOffset}
					seen[rr] = ri
					qi.rrOrder = append(qi.rrOrder, ri)
					qi.joins += " " + rr.joinSQL
					extraCols = append(extraCols, rr.cols...)
					extraOffset += len(rr.cols)
				}
				qi.rrByFT[f] = ri
			}

			qi.sel = "a.field, a.val, a.hidden, a.user_id, a.role"
			for _, c := range extraCols {
				qi.sel += ", " + c
			}
			batch.Queue(fmt.Sprintf(
				`SELECT %s FROM %s a%s WHERE a.%s = $1 AND a.field = ANY($2) AND a.pinned = true`,
				qi.sel, assertionsTable(ek.rt), qi.joins, entityIDCol(ek.rt)),
				ek.id, qi.fields)
			queries = append(queries, qi)
		}

		results := tx.SendBatch(ctx, batch)
		for _, qi := range queries {
			rows, err := results.Query()
			if err != nil {
				results.Close()
				return state, fmt.Errorf("fetchState: assertions: %w", err)
			}

			rs := state.records[qi.ek.id]

			type rawAssertion struct {
				val       json.RawMessage
				hidden    bool
				userID    *ID
				role      string
				extraCols []any
			}
			fieldRows := make(map[string][]rawAssertion)

			for rows.Next() {
				var field string
				var valJSON json.RawMessage
				var hidden bool
				var uid pgtype.UUID
				var rl pgtype.Text

				baseDests := []any{&field, &valJSON, &hidden, &uid, &rl}

				// Fresh scan destinations for extension columns, in JOIN order.
				var extraDests []any
				for _, ri := range qi.rrOrder {
					extraDests = append(extraDests, ri.rr.scanDests()...)
				}

				if err := rows.Scan(append(baseDests, extraDests...)...); err != nil {
					rows.Close()
					results.Close()
					return state, fmt.Errorf("fetchState: scan assertion: %w", err)
				}

				ra := rawAssertion{val: valJSON, hidden: hidden, extraCols: extraDests}
				if uid.Valid {
					id := ID(uid.Bytes)
					ra.userID = &id
				}
				if rl.Valid {
					ra.role = rl.String
				}
				fieldRows[field] = append(fieldRows[field], ra)
			}
			rows.Close()

			if rs == nil {
				continue
			}

			for field, raws := range fieldRows {
				first := raws[0]
				fs := &fieldState{
					hidden: first.hidden,
					userID: first.userID,
					role:   first.role,
				}

				if !first.hidden {
					ft, err := resolveFieldType(qi.ek.rt, field)
					if err == nil {
						ri := qi.rrByFT[field] // nil for non-FK fields

						if ft.collection {
							parts := make([]json.RawMessage, len(raws))
							for i, r := range raws {
								v := r.val
								if ri != nil {
									v = ri.rr.enrichVal(v, r.extraCols[ri.offset:ri.offset+len(ri.rr.cols)])
								}
								parts[i] = v
							}
							arrJSON, err := json.Marshal(parts)
							if err == nil {
								val, err := ft.unmarshal(json.RawMessage(arrJSON))
								if err == nil {
									fs.val = val
									rs.fields[field] = val
								}
							}
						} else {
							v := raws[0].val
							if ri != nil {
								v = ri.rr.enrichVal(v, raws[0].extraCols[ri.offset:ri.offset+len(ri.rr.cols)])
							}
							val, err := ft.unmarshal(v)
							if err == nil {
								fs.val = val
								rs.fields[field] = val
							}
						}
					}
				}

				rs.assertions[field] = fs
			}
		}
		if err := results.Close(); err != nil {
			return state, fmt.Errorf("fetchState: close assertions: %w", err)
		}
	}

	return state, nil
}

// nilIfEmpty returns nil for empty strings (for nullable text columns).
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
