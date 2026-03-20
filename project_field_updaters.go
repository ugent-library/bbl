package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for project assertion tables ---

func writeProjectAssertion(ctx context.Context, tx pgx.Tx, revID int64, projectID ID, field string, val any, hidden bool, projectSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeProjectAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_project_assertions (rev_id, project_id, field, val, hidden, project_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, projectID, field, valJSON, hidden, projectSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeProjectAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeProjectPerson(ctx context.Context, tx pgx.Tx, assertionID int64, personID ID, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_assertion_people (assertion_id, person_id, role)
		VALUES ($1, $2, $3)`,
		assertionID, personID, nilIfEmpty(role))
	if err != nil {
		return fmt.Errorf("writeProjectPerson: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset updaters for project collectives
// ============================================================

// --- SetProjectTitles (no delete — required) ---

type SetProjectTitles struct {
	ProjectID ID `json:"project_id"`
	Titles    []Title
	userID    *ID
	role      *string
}

func (m *SetProjectTitles) name() string       { return "set:project_titles" }
func (m *SetProjectTitles) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *SetProjectTitles) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && slicesEqual(p.Titles, m.Titles) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "titles") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "titles", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectTitles) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "titles", revID); err != nil {
		return fmt.Errorf("SetProjectTitles: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "titles"); err != nil {
		return fmt.Errorf("SetProjectTitles: %w", err)
	}
	for _, t := range m.Titles {
		if _, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "titles", t, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetProjectTitles: %w", err)
		}
	}
	return nil
}

// --- SetProjectDescriptions / UnsetProjectDescriptions ---

type SetProjectDescriptions struct {
	ProjectID    ID
	Descriptions []Text `json:"descriptions"`
	userID       *ID
	role         *string
}

func (m *SetProjectDescriptions) name() string       { return "set:project_descriptions" }
func (m *SetProjectDescriptions) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *SetProjectDescriptions) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && slicesEqual(p.Descriptions, m.Descriptions) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "descriptions") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "descriptions", revID); err != nil {
		return fmt.Errorf("SetProjectDescriptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "descriptions"); err != nil {
		return fmt.Errorf("SetProjectDescriptions: %w", err)
	}
	for _, t := range m.Descriptions {
		if _, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "descriptions", t, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetProjectDescriptions: %w", err)
		}
	}
	return nil
}

type UnsetProjectDescriptions struct{ ProjectID ID }

func (m *UnsetProjectDescriptions) name() string       { return "unset:project_descriptions" }
func (m *UnsetProjectDescriptions) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *UnsetProjectDescriptions) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && len(p.Descriptions) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "descriptions") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "descriptions", revID); err != nil {
		return fmt.Errorf("UnsetProjectDescriptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = 'descriptions' AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("UnsetProjectDescriptions: %w", err)
	}
	return nil
}

// --- SetProjectIdentifiers / UnsetProjectIdentifiers ---

type SetProjectIdentifiers struct {
	ProjectID   ID
	Identifiers []Identifier `json:"identifiers"`
	userID      *ID
	role        *string
}

func (m *SetProjectIdentifiers) name() string       { return "set:project_identifiers" }
func (m *SetProjectIdentifiers) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *SetProjectIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && slicesEqual(p.Identifiers, m.Identifiers) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "identifiers", revID); err != nil {
		return fmt.Errorf("SetProjectIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "identifiers"); err != nil {
		return fmt.Errorf("SetProjectIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if _, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "identifiers", ident, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetProjectIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetProjectIdentifiers struct{ ProjectID ID }

func (m *UnsetProjectIdentifiers) name() string       { return "unset:project_identifiers" }
func (m *UnsetProjectIdentifiers) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *UnsetProjectIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && len(p.Identifiers) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "identifiers", revID); err != nil {
		return fmt.Errorf("UnsetProjectIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("UnsetProjectIdentifiers: %w", err)
	}
	return nil
}

// --- SetProjectPeople / UnsetProjectPeople ---

type SetProjectPeople struct {
	ProjectID ID `json:"project_id"`
	People    []ProjectPerson
	userID    *ID
	role      *string
}

func (m *SetProjectPeople) name() string       { return "set:project_people" }
func (m *SetProjectPeople) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *SetProjectPeople) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && projectPeopleEqual(p.People, m.People) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "people") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "people", revID); err != nil {
		return fmt.Errorf("SetProjectPeople: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "people"); err != nil {
		return fmt.Errorf("SetProjectPeople: %w", err)
	}
	for _, p := range m.People {
		val := struct {
			Role string `json:"role,omitempty"`
		}{p.Role}
		assertionID, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "people", val, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetProjectPeople: %w", err)
		}
		if err := writeProjectPerson(ctx, tx, assertionID, p.PersonID, p.Role); err != nil {
			return fmt.Errorf("SetProjectPeople: %w", err)
		}
	}
	return nil
}

type UnsetProjectPeople struct{ ProjectID ID }

func (m *UnsetProjectPeople) name() string       { return "unset:project_people" }
func (m *UnsetProjectPeople) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *UnsetProjectPeople) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.projects[m.ProjectID]; p != nil && len(p.People) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "people") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "people", revID); err != nil {
		return fmt.Errorf("UnsetProjectPeople: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = 'people' AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("UnsetProjectPeople: %w", err)
	}
	return nil
}

// ============================================================
// Hide updaters for project fields
// ============================================================

// --- HideProjectDescriptions ---

type HideProjectDescriptions struct {
	ProjectID ID
	userID    *ID
	role      *string
}

func (m *HideProjectDescriptions) name() string       { return "hide:project_descriptions" }
func (m *HideProjectDescriptions) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *HideProjectDescriptions) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.projectAssertions[m.ProjectID], "descriptions") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "descriptions") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "descriptions", revID); err != nil {
		return fmt.Errorf("HideProjectDescriptions: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "descriptions"); err != nil {
		return fmt.Errorf("HideProjectDescriptions: %w", err)
	}
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "descriptions", nil, true, nil, m.userID, m.role)
	return err
}

// --- HideProjectIdentifiers ---

type HideProjectIdentifiers struct {
	ProjectID ID
	userID    *ID
	role      *string
}

func (m *HideProjectIdentifiers) name() string       { return "hide:project_identifiers" }
func (m *HideProjectIdentifiers) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *HideProjectIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.projectAssertions[m.ProjectID], "identifiers") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "identifiers", revID); err != nil {
		return fmt.Errorf("HideProjectIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "identifiers"); err != nil {
		return fmt.Errorf("HideProjectIdentifiers: %w", err)
	}
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "identifiers", nil, true, nil, m.userID, m.role)
	return err
}

// --- HideProjectPeople ---

type HideProjectPeople struct {
	ProjectID ID
	userID    *ID
	role      *string
}

func (m *HideProjectPeople) name() string       { return "hide:project_people" }
func (m *HideProjectPeople) needs() updateNeeds { return updateNeeds{projectIDs: []ID{m.ProjectID}} }
func (m *HideProjectPeople) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.projectAssertions[m.ProjectID], "people") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.projectAssertions[m.ProjectID], "people") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logProjectHistory(ctx, tx, m.ProjectID, "people", revID); err != nil {
		return fmt.Errorf("HideProjectPeople: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_assertions WHERE project_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.ProjectID, "people"); err != nil {
		return fmt.Errorf("HideProjectPeople: %w", err)
	}
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "people", nil, true, nil, m.userID, m.role)
	return err
}
