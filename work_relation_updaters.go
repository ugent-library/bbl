package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeWorkAssertion creates an assertion row. Used by both Set updaters and import.
func writeWorkAssertion(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, hidden bool, workSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, hidden, work_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, workID, field, valJSON, hidden, workSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
	}
	return id, nil
}

// --- extension table helpers for FK-bearing collectives ---

func writeWorkContributor(ctx context.Context, tx pgx.Tx, assertionID int64, personID *ID, organizationID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_contributors (assertion_id, person_id, organization_id)
		VALUES ($1, $2, $3)`,
		assertionID, personID, organizationID)
	if err != nil {
		return fmt.Errorf("writeWorkContributor: %w", err)
	}
	return nil
}

func writeWorkProject(ctx context.Context, tx pgx.Tx, assertionID int64, projectID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_projects (assertion_id, project_id)
		VALUES ($1, $2)`,
		assertionID, projectID)
	if err != nil {
		return fmt.Errorf("writeWorkProject: %w", err)
	}
	return nil
}

func writeWorkOrganization(ctx context.Context, tx pgx.Tx, assertionID int64, organizationID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_organizations (assertion_id, organization_id)
		VALUES ($1, $2)`,
		assertionID, organizationID)
	if err != nil {
		return fmt.Errorf("writeWorkOrganization: %w", err)
	}
	return nil
}

func writeWorkRel(ctx context.Context, tx pgx.Tx, assertionID int64, relatedWorkID ID, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertion_rels (assertion_id, related_work_id, kind)
		VALUES ($1, $2, $3)`,
		assertionID, relatedWorkID, kind)
	if err != nil {
		return fmt.Errorf("writeWorkRel: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset updaters for work collectives
// ============================================================

// --- SetWorkTitles (no delete — required) ---

type SetWorkTitles struct {
	WorkID ID      `json:"work_id"`
	Titles []Title `json:"titles"`
	userID *ID
	role   *string
}

func (m *SetWorkTitles) name() string       { return "set:work_titles" }
func (m *SetWorkTitles) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkTitles) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Titles, m.Titles) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "titles") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "titles", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkTitles) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "titles", revID); err != nil {
		return fmt.Errorf("SetWorkTitles: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'titles' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkTitles: %w", err)
	}
	for _, t := range m.Titles {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "titles", t, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkTitles: %w", err)
		}
	}
	return nil
}

// --- SetWorkAbstracts / UnsetWorkAbstracts ---

type SetWorkAbstracts struct {
	WorkID    ID
	Abstracts []Text `json:"abstracts"`
	userID    *ID
	role      *string
}

func (m *SetWorkAbstracts) name() string       { return "set:work_abstracts" }
func (m *SetWorkAbstracts) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkAbstracts) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Abstracts, m.Abstracts) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "abstracts") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "abstracts", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkAbstracts) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "abstracts", revID); err != nil {
		return fmt.Errorf("SetWorkAbstracts: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'abstracts' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkAbstracts: %w", err)
	}
	for _, t := range m.Abstracts {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "abstracts", t, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkAbstracts: %w", err)
		}
	}
	return nil
}

type UnsetWorkAbstracts struct{ WorkID ID }

func (m *UnsetWorkAbstracts) name() string       { return "unset:work_abstracts" }
func (m *UnsetWorkAbstracts) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkAbstracts) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Abstracts) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "abstracts") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "abstracts", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkAbstracts) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "abstracts", revID); err != nil {
		return fmt.Errorf("UnsetWorkAbstracts: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'abstracts' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkAbstracts: %w", err)
	}
	return nil
}

// --- SetWorkLaySummaries / UnsetWorkLaySummaries ---

type SetWorkLaySummaries struct {
	WorkID       ID
	LaySummaries []Text `json:"lay_summaries"`
	userID       *ID
	role         *string
}

func (m *SetWorkLaySummaries) name() string       { return "set:work_lay_summaries" }
func (m *SetWorkLaySummaries) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkLaySummaries) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.LaySummaries, m.LaySummaries) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "lay_summaries") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "lay_summaries", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkLaySummaries) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "lay_summaries", revID); err != nil {
		return fmt.Errorf("SetWorkLaySummaries: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'lay_summaries' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkLaySummaries: %w", err)
	}
	for _, t := range m.LaySummaries {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "lay_summaries", t, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkLaySummaries: %w", err)
		}
	}
	return nil
}

type UnsetWorkLaySummaries struct{ WorkID ID }

func (m *UnsetWorkLaySummaries) name() string       { return "unset:work_lay_summaries" }
func (m *UnsetWorkLaySummaries) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkLaySummaries) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.LaySummaries) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "lay_summaries") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "lay_summaries", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkLaySummaries) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "lay_summaries", revID); err != nil {
		return fmt.Errorf("UnsetWorkLaySummaries: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'lay_summaries' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkLaySummaries: %w", err)
	}
	return nil
}

// --- SetWorkNotes / UnsetWorkNotes ---

type SetWorkNotes struct {
	WorkID ID     `json:"work_id"`
	Notes  []Note `json:"notes"`
	userID *ID
	role   *string
}

func (m *SetWorkNotes) name() string       { return "set:work_notes" }
func (m *SetWorkNotes) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkNotes) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Notes, m.Notes) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "notes") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "notes", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkNotes) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "notes", revID); err != nil {
		return fmt.Errorf("SetWorkNotes: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'notes' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkNotes: %w", err)
	}
	for _, n := range m.Notes {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "notes", n, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkNotes: %w", err)
		}
	}
	return nil
}

type UnsetWorkNotes struct{ WorkID ID }

func (m *UnsetWorkNotes) name() string       { return "unset:work_notes" }
func (m *UnsetWorkNotes) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkNotes) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Notes) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "notes") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "notes", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkNotes) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "notes", revID); err != nil {
		return fmt.Errorf("UnsetWorkNotes: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'notes' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkNotes: %w", err)
	}
	return nil
}

// --- SetWorkKeywords / UnsetWorkKeywords ---

type SetWorkKeywords struct {
	WorkID   ID
	Keywords []Keyword `json:"keywords"`
	userID   *ID
	role     *string
}

func (m *SetWorkKeywords) name() string       { return "set:work_keywords" }
func (m *SetWorkKeywords) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkKeywords) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Keywords, m.Keywords) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "keywords") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "keywords", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkKeywords) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "keywords", revID); err != nil {
		return fmt.Errorf("SetWorkKeywords: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'keywords' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkKeywords: %w", err)
	}
	for _, k := range m.Keywords {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "keywords", k, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkKeywords: %w", err)
		}
	}
	return nil
}

type UnsetWorkKeywords struct{ WorkID ID }

func (m *UnsetWorkKeywords) name() string       { return "unset:work_keywords" }
func (m *UnsetWorkKeywords) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkKeywords) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Keywords) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "keywords") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "keywords", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkKeywords) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "keywords", revID); err != nil {
		return fmt.Errorf("UnsetWorkKeywords: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'keywords' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkKeywords: %w", err)
	}
	return nil
}

// --- SetWorkIdentifiers / UnsetWorkIdentifiers ---

type SetWorkIdentifiers struct {
	WorkID      ID
	Identifiers []WorkIdentifier
	userID      *ID
	role        *string
}

func (m *SetWorkIdentifiers) name() string       { return "set:work_identifiers" }
func (m *SetWorkIdentifiers) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Identifiers, m.Identifiers) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "identifiers", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "identifiers", revID); err != nil {
		return fmt.Errorf("SetWorkIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "identifiers", ident, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetWorkIdentifiers struct{ WorkID ID }

func (m *UnsetWorkIdentifiers) name() string       { return "unset:work_identifiers" }
func (m *UnsetWorkIdentifiers) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Identifiers) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "identifiers", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "identifiers", revID); err != nil {
		return fmt.Errorf("UnsetWorkIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkIdentifiers: %w", err)
	}
	return nil
}

// --- SetWorkClassifications / UnsetWorkClassifications ---

type SetWorkClassifications struct {
	WorkID          ID
	Classifications []WorkClassification
	userID          *ID
	role            *string
}

func (m *SetWorkClassifications) name() string       { return "set:work_classifications" }
func (m *SetWorkClassifications) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkClassifications) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Classifications, m.Classifications) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "classifications") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "classifications", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkClassifications) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "classifications", revID); err != nil {
		return fmt.Errorf("SetWorkClassifications: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'classifications' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkClassifications: %w", err)
	}
	for _, c := range m.Classifications {
		if _, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "classifications", c, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetWorkClassifications: %w", err)
		}
	}
	return nil
}

type UnsetWorkClassifications struct{ WorkID ID }

func (m *UnsetWorkClassifications) name() string       { return "unset:work_classifications" }
func (m *UnsetWorkClassifications) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkClassifications) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Classifications) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "classifications") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "classifications", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkClassifications) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "classifications", revID); err != nil {
		return fmt.Errorf("UnsetWorkClassifications: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'classifications' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkClassifications: %w", err)
	}
	return nil
}

// --- SetWorkContributors / UnsetWorkContributors ---

type SetWorkContributors struct {
	WorkID       ID
	Contributors []WorkContributor `json:"contributors"`
	userID       *ID
	role         *string
}

func (m *SetWorkContributors) name() string       { return "set:work_contributors" }
func (m *SetWorkContributors) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkContributors) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && contributorsEqual(w.Contributors, m.Contributors) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "contributors") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "contributors", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkContributors) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "contributors", revID); err != nil {
		return fmt.Errorf("SetWorkContributors: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'contributors' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkContributors: %w", err)
	}
	for _, c := range m.Contributors {
		val := struct {
			Kind       string   `json:"kind,omitempty"`
			Name       string   `json:"name"`
			GivenName  string   `json:"given_name,omitempty"`
			FamilyName string   `json:"family_name,omitempty"`
			Roles      []string `json:"roles,omitempty"`
		}{c.Kind, c.Name, c.GivenName, c.FamilyName, c.Roles}
		assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "contributors", val, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetWorkContributors: %w", err)
		}
		if err := writeWorkContributor(ctx, tx, assertionID, c.PersonID, nil); err != nil {
			return fmt.Errorf("SetWorkContributors: %w", err)
		}
	}
	return nil
}

type UnsetWorkContributors struct{ WorkID ID }

func (m *UnsetWorkContributors) name() string       { return "unset:work_contributors" }
func (m *UnsetWorkContributors) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkContributors) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Contributors) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "contributors") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "contributors", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkContributors) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "contributors", revID); err != nil {
		return fmt.Errorf("UnsetWorkContributors: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'contributors' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkContributors: %w", err)
	}
	return nil
}

// --- SetWorkProjects / UnsetWorkProjects ---

type SetWorkProjects struct {
	WorkID   ID
	Projects []ID `json:"projects"`
	userID   *ID
	role     *string
}

func (m *SetWorkProjects) name() string       { return "set:work_projects" }
func (m *SetWorkProjects) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkProjects) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Projects, m.Projects) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "projects") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "projects", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkProjects) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "projects", revID); err != nil {
		return fmt.Errorf("SetWorkProjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'projects' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkProjects: %w", err)
	}
	for _, pid := range m.Projects {
		assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "projects", nil, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetWorkProjects: %w", err)
		}
		if err := writeWorkProject(ctx, tx, assertionID, pid); err != nil {
			return fmt.Errorf("SetWorkProjects: %w", err)
		}
	}
	return nil
}

type UnsetWorkProjects struct{ WorkID ID }

func (m *UnsetWorkProjects) name() string       { return "unset:work_projects" }
func (m *UnsetWorkProjects) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkProjects) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Projects) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "projects") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "projects", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkProjects) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "projects", revID); err != nil {
		return fmt.Errorf("UnsetWorkProjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'projects' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkProjects: %w", err)
	}
	return nil
}

// --- SetWorkOrganizations / UnsetWorkOrganizations ---

type SetWorkOrganizations struct {
	WorkID        ID
	Organizations []ID `json:"organizations"`
	userID        *ID
	role          *string
}

func (m *SetWorkOrganizations) name() string       { return "set:work_organizations" }
func (m *SetWorkOrganizations) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkOrganizations) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && slicesEqual(w.Organizations, m.Organizations) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "organizations") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "organizations", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "organizations", revID); err != nil {
		return fmt.Errorf("SetWorkOrganizations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkOrganizations: %w", err)
	}
	for _, oid := range m.Organizations {
		assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "organizations", nil, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetWorkOrganizations: %w", err)
		}
		if err := writeWorkOrganization(ctx, tx, assertionID, oid); err != nil {
			return fmt.Errorf("SetWorkOrganizations: %w", err)
		}
	}
	return nil
}

type UnsetWorkOrganizations struct{ WorkID ID }

func (m *UnsetWorkOrganizations) name() string       { return "unset:work_organizations" }
func (m *UnsetWorkOrganizations) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkOrganizations) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Organizations) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "organizations") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "organizations", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "organizations", revID); err != nil {
		return fmt.Errorf("UnsetWorkOrganizations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkOrganizations: %w", err)
	}
	return nil
}

// --- SetWorkRels / UnsetWorkRels ---

type SetWorkRels struct {
	WorkID ID `json:"work_id"`
	Rels   []struct {
		RelatedWorkID ID     `json:"related_work_id"`
		Kind          string `json:"kind"`
	} `json:"rels"`
	userID *ID
	role   *string
}

func (m *SetWorkRels) name() string       { return "set:work_rels" }
func (m *SetWorkRels) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkRels) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && workRelsMatch(w.Rels, m.Rels) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "rels") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "rels", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "rels", revID); err != nil {
		return fmt.Errorf("SetWorkRels: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkRels: %w", err)
	}
	for _, r := range m.Rels {
		val := struct {
			Kind string `json:"kind"`
		}{r.Kind}
		assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "rels", val, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetWorkRels: %w", err)
		}
		if err := writeWorkRel(ctx, tx, assertionID, r.RelatedWorkID, r.Kind); err != nil {
			return fmt.Errorf("SetWorkRels: %w", err)
		}
	}
	return nil
}

type UnsetWorkRels struct{ WorkID ID }

func (m *UnsetWorkRels) name() string       { return "unset:work_rels" }
func (m *UnsetWorkRels) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkRels) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && len(w.Rels) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[m.WorkID], "rels") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "rels", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logWorkHistory(ctx, tx, m.WorkID, "rels", revID); err != nil {
		return fmt.Errorf("UnsetWorkRels: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkRels: %w", err)
	}
	return nil
}
