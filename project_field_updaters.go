package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for project relation tables ---

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

func writeProjectTitle(ctx context.Context, tx pgx.Tx, id, projectID ID, assertionID int64, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_titles (id, assertion_id, project_id, lang, val)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'und'), $5)`,
		id, assertionID, projectID, lang, val)
	if err != nil {
		return fmt.Errorf("writeProjectTitle: %w", err)
	}
	return nil
}

func writeProjectDescription(ctx context.Context, tx pgx.Tx, id, projectID ID, assertionID int64, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_descriptions (id, assertion_id, project_id, lang, val)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'und'), $5)`,
		id, assertionID, projectID, lang, val)
	if err != nil {
		return fmt.Errorf("writeProjectDescription: %w", err)
	}
	return nil
}

func writeProjectIdentifier(ctx context.Context, tx pgx.Tx, id, projectID ID, assertionID int64, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_identifiers (id, assertion_id, project_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, projectID, scheme, val)
	if err != nil {
		return fmt.Errorf("writeProjectIdentifier: %w", err)
	}
	return nil
}

func writeProjectPerson(ctx context.Context, tx pgx.Tx, id, projectID, personID ID, assertionID int64, role string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_people (id, assertion_id, project_id, person_id, role)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, projectID, personID, nilIfEmpty(role))
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
}

func (m *SetProjectTitles) name() string       { return "set:project_titles" }
func (m *SetProjectTitles) needs() updateNeeds { return updateNeeds{} }
func (m *SetProjectTitles) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "titles", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectTitles) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "titles", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetProjectTitles: %w", err)
	}
	for _, t := range m.Titles {
		if err := writeProjectTitle(ctx, tx, newID(), m.ProjectID, assertionID, t.Lang, t.Val); err != nil {
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
}

func (m *SetProjectDescriptions) name() string       { return "set:project_descriptions" }
func (m *SetProjectDescriptions) needs() updateNeeds { return updateNeeds{} }
func (m *SetProjectDescriptions) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "descriptions", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetProjectDescriptions: %w", err)
	}
	for _, t := range m.Descriptions {
		if err := writeProjectDescription(ctx, tx, newID(), m.ProjectID, assertionID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetProjectDescriptions: %w", err)
		}
	}
	return nil
}

type UnsetProjectDescriptions struct{ ProjectID ID }

func (m *UnsetProjectDescriptions) name() string       { return "unset:project_descriptions" }
func (m *UnsetProjectDescriptions) needs() updateNeeds { return updateNeeds{} }
func (m *UnsetProjectDescriptions) apply(state updateState, userID *ID) (*updateEffect, error) {
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
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
}

func (m *SetProjectIdentifiers) name() string       { return "set:project_identifiers" }
func (m *SetProjectIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *SetProjectIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "identifiers", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetProjectIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeProjectIdentifier(ctx, tx, newID(), m.ProjectID, assertionID, ident.Scheme, ident.Val); err != nil {
			return fmt.Errorf("SetProjectIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetProjectIdentifiers struct{ ProjectID ID }

func (m *UnsetProjectIdentifiers) name() string       { return "unset:project_identifiers" }
func (m *UnsetProjectIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *UnsetProjectIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
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
}

func (m *SetProjectPeople) name() string       { return "set:project_people" }
func (m *SetProjectPeople) needs() updateNeeds { return updateNeeds{} }
func (m *SetProjectPeople) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "people", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetProjectPeople: %w", err)
	}
	for _, p := range m.People {
		if err := writeProjectPerson(ctx, tx, newID(), m.ProjectID, p.PersonID, assertionID, p.Role); err != nil {
			return fmt.Errorf("SetProjectPeople: %w", err)
		}
	}
	return nil
}

type UnsetProjectPeople struct{ ProjectID ID }

func (m *UnsetProjectPeople) name() string       { return "unset:project_people" }
func (m *UnsetProjectPeople) needs() updateNeeds { return updateNeeds{} }
func (m *UnsetProjectPeople) apply(state updateState, userID *ID) (*updateEffect, error) {
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *UnsetProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
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
}

func (m *HideProjectDescriptions) name() string       { return "hide:project_descriptions" }
func (m *HideProjectDescriptions) needs() updateNeeds { return updateNeeds{} }
func (m *HideProjectDescriptions) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "descriptions", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectDescriptions) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "descriptions", nil, true, nil, m.userID, nil)
	return err
}

// --- HideProjectIdentifiers ---

type HideProjectIdentifiers struct {
	ProjectID ID
	userID    *ID
}

func (m *HideProjectIdentifiers) name() string       { return "hide:project_identifiers" }
func (m *HideProjectIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *HideProjectIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "identifiers", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "identifiers", nil, true, nil, m.userID, nil)
	return err
}

// --- HideProjectPeople ---

type HideProjectPeople struct {
	ProjectID ID
	userID    *ID
}

func (m *HideProjectPeople) name() string       { return "hide:project_people" }
func (m *HideProjectPeople) needs() updateNeeds { return updateNeeds{} }
func (m *HideProjectPeople) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_project_assertions", "project_id", m.ProjectID, "people", "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *HideProjectPeople) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	_, err := writeProjectAssertion(ctx, tx, revID, m.ProjectID, "people", nil, true, nil, m.userID, nil)
	return err
}
