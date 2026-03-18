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
// Set / Unset mutations for project collectives
// ============================================================

// --- SetProjectTitles (no delete — required) ---

type SetProjectTitles struct {
	ProjectID ID     `json:"project_id"`
	Titles    []Title
	userID    *ID
}

func (m *SetProjectTitles) mutationName() string { return "set_project_titles" }
func (m *SetProjectTitles) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectTitles) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
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

func (m *SetProjectDescriptions) mutationName() string { return "set_project_descriptions" }
func (m *SetProjectDescriptions) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectDescriptions) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
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

func (m *UnsetProjectDescriptions) mutationName() string { return "unset_project_descriptions" }
func (m *UnsetProjectDescriptions) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetProjectDescriptions) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
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

func (m *SetProjectIdentifiers) mutationName() string { return "set_project_identifiers" }
func (m *SetProjectIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
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

func (m *UnsetProjectIdentifiers) mutationName() string { return "unset_project_identifiers" }
func (m *UnsetProjectIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetProjectIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
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
	ProjectID ID     `json:"project_id"`
	People    []ProjectPerson
	userID    *ID
}

func (m *SetProjectPeople) mutationName() string { return "set_project_people" }
func (m *SetProjectPeople) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectPeople) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
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

func (m *UnsetProjectPeople) mutationName() string { return "unset_project_people" }
func (m *UnsetProjectPeople) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetProjectPeople) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
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
