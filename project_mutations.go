package bbl

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateProject creates a new project entity.
type CreateProject struct {
	ProjectID  ID     `json:"project_id"`
	Status    string     // defaults to ProjectStatusPublic
	StartDate *time.Time
	EndDate   *time.Time

	project *Project // populated by apply
}

func (m *CreateProject) mutationName() string { return "create_project" }

func (m *CreateProject) needs() mutationNeeds { return mutationNeeds{} }

func (m *CreateProject) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	if m.Status == "" {
		m.Status = ProjectStatusPublic
	}
	m.project = &Project{
		ID:        m.ProjectID,
		Version:   1,
		Status:    m.Status,
		StartDate: m.StartDate,
		EndDate:   m.EndDate,
	}
	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		record:     m.project,
	}, nil
}

func (m *CreateProject) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	p := m.project
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_projects
		    (id, version, created_by_id, updated_by_id, status,
		     start_date, end_date, deleted_at, deleted_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		p.ID, p.Version, p.CreatedByID, p.UpdatedByID, p.Status,
		p.StartDate, p.EndDate, p.DeletedAt, p.DeletedByID)
	if err != nil {
		return fmt.Errorf("CreateProject.write: %w", err)
	}
	return nil
}

// DeleteProject soft-deletes a project.
type DeleteProject struct {
	ProjectID ID     `json:"project_id"`

	project *Project // populated by apply
}

func (m *DeleteProject) mutationName() string { return "delete_project" }

func (m *DeleteProject) needs() mutationNeeds {
	return mutationNeeds{projectIDs: []ID{m.ProjectID}}
}

func (m *DeleteProject) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	p, ok := state.projects[m.ProjectID]
	if !ok {
		return nil, fmt.Errorf("DeleteProject: project %s not found", m.ProjectID)
	}
	if p.Status == ProjectStatusDeleted {
		return nil, nil // noop
	}

	now := time.Now()
	p.Version++
	p.Status = ProjectStatusDeleted
	p.DeletedAt = &now
	m.project = p

	return &mutationEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
		record:     p,
	}, nil
}

func (m *DeleteProject) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	p := m.project
	_, err := tx.Exec(ctx, `
		UPDATE bbl_projects
		SET version = $2, updated_at = transaction_timestamp(),
		    updated_by_id = $3, status = $4,
		    deleted_at = $5, deleted_by_id = $6
		WHERE id = $1`,
		p.ID, p.Version, p.UpdatedByID,
		p.Status, p.DeletedAt, p.DeletedByID)
	if err != nil {
		return fmt.Errorf("DeleteProject.write: %w", err)
	}
	return nil
}
