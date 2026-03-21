package bbl

import "fmt"

// CreateWork creates a new work entity.
type CreateWork struct {
	ID     ID     `json:"id"`
	Kind   string `json:"kind"`
	Status string `json:"status"` // defaults to WorkStatusPrivate
}

func (m *CreateWork) name() string { return "create:work" }

func (m *CreateWork) needs() updateNeeds { return updateNeeds{} }

func (m *CreateWork) apply(state updateState, user *User) (*updateEffect, error) {
	if m.Status == "" {
		m.Status = WorkStatusPrivate
	}
	state.records[m.ID] = &recordState{
		recordType: RecordTypeWork,
		id:         m.ID,
		status:     m.Status,
		kind:       m.Kind,
		fields:     make(map[string]any),
		assertions: make(map[string]*fieldState),
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.ID,
	}, nil
}

func (m *CreateWork) write(revID int64, user *User) (string, []any) {
	return `INSERT INTO bbl_works
		    (id, version, created_by_id, updated_by_id, kind, status)
		VALUES ($1, 1, $2, $3, $4, $5)`,
		[]any{m.ID, &user.ID, &user.ID, m.Kind, m.Status}
}

// DeleteWork soft-deletes a work.
type DeleteWork struct {
	WorkID     ID     `json:"id"`
	DeleteKind string // withdrawn, retracted, takedown
}

func (m *DeleteWork) name() string { return "delete:work" }

func (m *DeleteWork) needs() updateNeeds {
	return updateNeeds{workIDs: []ID{m.WorkID}}
}

func (m *DeleteWork) apply(state updateState, user *User) (*updateEffect, error) {
	rs := state.records[m.WorkID]
	if rs == nil {
		return nil, fmt.Errorf("DeleteWork: work %s not found", m.WorkID)
	}
	if rs.status == WorkStatusDeleted {
		return nil, nil // noop
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
	}, nil
}

func (m *DeleteWork) write(revID int64, user *User) (string, []any) {
	return `UPDATE bbl_works
		SET status = $2, delete_kind = $3,
		    deleted_at = transaction_timestamp(), deleted_by_id = $4
		WHERE id = $1`,
		[]any{m.WorkID, WorkStatusDeleted, nilIfEmpty(m.DeleteKind), &user.ID}
}
