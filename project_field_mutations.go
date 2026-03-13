package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for project relation tables ---
// These are used by both Set mutations (human path) and import.

func writeProjectTitle(ctx context.Context, tx pgx.Tx, id, projectID ID, lang, val string, projectSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_titles (id, project_id, lang, val, project_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, lang, val, projectSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeProjectTitle: %w", err)
	}
	return nil
}

func writeProjectDescription(ctx context.Context, tx pgx.Tx, id, projectID ID, lang, val string, projectSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_descriptions (id, project_id, lang, val, project_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, lang, val, projectSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeProjectDescription: %w", err)
	}
	return nil
}

func writeProjectIdentifier(ctx context.Context, tx pgx.Tx, id, projectID ID, scheme, val string, projectSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_identifiers (id, project_id, scheme, val, project_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, scheme, val, projectSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeProjectIdentifier: %w", err)
	}
	return nil
}

func writeProjectPerson(ctx context.Context, tx pgx.Tx, id, projectID, personID ID, role string, projectSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_project_people (id, project_id, person_id, role, project_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, projectID, personID, nilIfEmpty(role), projectSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeProjectPerson: %w", err)
	}
	return nil
}

// ============================================================
// Set / Delete mutations for project collectives
// ============================================================

// --- SetProjectTitles (no delete — required) ---

type SetProjectTitles struct {
	ProjectID ID
	Titles    []Title
	userID    *ID
}

func (m *SetProjectTitles) mutationName() string { return "SetProjectTitles" }
func (m *SetProjectTitles) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectTitles) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Titles []Title }{m.Titles}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_titles", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectTitles) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_titles WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("SetProjectTitles: delete: %w", err)
	}
	for _, t := range m.Titles {
		if err := writeProjectTitle(ctx, tx, newID(), m.ProjectID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetProjectTitles: %w", err)
		}
	}
	return nil
}

// --- SetProjectDescriptions / DeleteProjectDescriptions ---

type SetProjectDescriptions struct {
	ProjectID    ID
	Descriptions []Text
	userID       *ID
}

func (m *SetProjectDescriptions) mutationName() string { return "SetProjectDescriptions" }
func (m *SetProjectDescriptions) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectDescriptions) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Descriptions []Text }{m.Descriptions}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_descriptions", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectDescriptions) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_descriptions WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("SetProjectDescriptions: delete: %w", err)
	}
	for _, t := range m.Descriptions {
		if err := writeProjectDescription(ctx, tx, newID(), m.ProjectID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetProjectDescriptions: %w", err)
		}
	}
	return nil
}

type DeleteProjectDescriptions struct{ ProjectID ID }

func (m *DeleteProjectDescriptions) mutationName() string { return "DeleteProjectDescriptions" }
func (m *DeleteProjectDescriptions) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteProjectDescriptions) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_descriptions", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *DeleteProjectDescriptions) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_descriptions WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("DeleteProjectDescriptions: %w", err)
	}
	return nil
}

// --- SetProjectIdentifiers / DeleteProjectIdentifiers ---

type SetProjectIdentifiers struct {
	ProjectID   ID
	Identifiers []Identifier
	userID      *ID
}

func (m *SetProjectIdentifiers) mutationName() string { return "SetProjectIdentifiers" }
func (m *SetProjectIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Identifiers []Identifier }{m.Identifiers}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_identifiers", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_identifiers WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("SetProjectIdentifiers: delete: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeProjectIdentifier(ctx, tx, newID(), m.ProjectID, ident.Scheme, ident.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetProjectIdentifiers: %w", err)
		}
	}
	return nil
}

type DeleteProjectIdentifiers struct{ ProjectID ID }

func (m *DeleteProjectIdentifiers) mutationName() string { return "DeleteProjectIdentifiers" }
func (m *DeleteProjectIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteProjectIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_identifiers", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *DeleteProjectIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_identifiers WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("DeleteProjectIdentifiers: %w", err)
	}
	return nil
}

// --- SetProjectPeople / DeleteProjectPeople ---

type SetProjectPeople struct {
	ProjectID ID
	People    []ProjectPerson
	userID    *ID
}

func (m *SetProjectPeople) mutationName() string { return "SetProjectPeople" }
func (m *SetProjectPeople) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetProjectPeople) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ People []ProjectPerson }{m.People}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_people", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *SetProjectPeople) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_people WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("SetProjectPeople: delete: %w", err)
	}
	for _, p := range m.People {
		if err := writeProjectPerson(ctx, tx, newID(), m.ProjectID, p.PersonID, p.Role, nil, m.userID); err != nil {
			return fmt.Errorf("SetProjectPeople: %w", err)
		}
	}
	return nil
}

type DeleteProjectPeople struct{ ProjectID ID }

func (m *DeleteProjectPeople) mutationName() string { return "DeleteProjectPeople" }
func (m *DeleteProjectPeople) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteProjectPeople) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_project_people", "project_id", m.ProjectID, "project_source_id", "bbl_project_sources", priorities)
		},
	}, nil
}
func (m *DeleteProjectPeople) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_project_people WHERE project_id = $1 AND user_id IS NOT NULL`, m.ProjectID); err != nil {
		return fmt.Errorf("DeleteProjectPeople: %w", err)
	}
	return nil
}
