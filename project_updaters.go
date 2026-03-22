package bbl

import (
	"fmt"
	"time"
)

// CreateProject creates a new project entity.
type CreateProject struct {
	ID        ID     `json:"id"`
	Status    string // defaults to ProjectStatusPublic
	StartDate *time.Time
	EndDate   *time.Time
}

func (m *CreateProject) name() string { return "create:project" }

func (m *CreateProject) needs() updateNeeds { return updateNeeds{} }

func (m *CreateProject) apply(state updateState, user *User) (*updateEffect, error) {
	if m.Status == "" {
		m.Status = ProjectStatusPublic
	}
	state.records[m.ID] = &recordState{
		recordType: RecordTypeProject,
		id:         m.ID,
		status:     m.Status,
		fields:     make(map[string]any),
		assertions: make(map[string][]assertion),
	}
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ID,
	}, nil
}

func (m *CreateProject) write(revID int64, user *User) (string, []any) {
	return `INSERT INTO bbl_projects
		    (id, version, created_by_id, updated_by_id, status,
		     start_date, end_date)
		VALUES ($1, 1, $2, $3, $4, $5, $6)`,
		[]any{m.ID, &user.ID, &user.ID, m.Status, m.StartDate, m.EndDate}
}

// DeleteProject soft-deletes a project.
type DeleteProject struct {
	ProjectID ID `json:"id"`
}

func (m *DeleteProject) name() string { return "delete:project" }

func (m *DeleteProject) needs() updateNeeds {
	return updateNeeds{projectIDs: []ID{m.ProjectID}}
}

func (m *DeleteProject) apply(state updateState, user *User) (*updateEffect, error) {
	rs := state.records[m.ProjectID]
	if rs == nil {
		return nil, fmt.Errorf("DeleteProject: project %s not found", m.ProjectID)
	}
	if rs.status == ProjectStatusDeleted {
		return nil, nil // noop
	}
	return &updateEffect{
		recordType: RecordTypeProject,
		recordID:   m.ProjectID,
	}, nil
}

func (m *DeleteProject) write(revID int64, user *User) (string, []any) {
	return `UPDATE bbl_projects
		SET status = $2,
		    deleted_at = transaction_timestamp(), deleted_by_id = $3
		WHERE id = $1`,
		[]any{m.ProjectID, ProjectStatusDeleted, &user.ID}
}
