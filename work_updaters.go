package bbl

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateWork creates a new work entity.
type CreateWork struct {
	WorkID ID     `json:"work_id"`
	Kind   string `json:"kind"`
	Status string `json:"status"` // defaults to WorkStatusPrivate

	work *Work // populated by apply
}

func (m *CreateWork) name() string { return "create:work" }

func (m *CreateWork) needs() updateNeeds { return updateNeeds{} }

func (m *CreateWork) apply(state updateState, userID *ID) (*updateEffect, error) {
	if m.Status == "" {
		m.Status = WorkStatusPrivate
	}
	m.work = &Work{
		ID:      m.WorkID,
		Version: 1,
		Kind:    m.Kind,
		Status:  m.Status,
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		record:     m.work,
	}, nil
}

func (m *CreateWork) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	w := m.work
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_works
		    (id, version, created_by_id, updated_by_id,
		     kind, status, review_status, delete_kind,
		     deleted_at, deleted_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		w.ID, w.Version, w.CreatedByID, w.UpdatedByID,
		w.Kind, w.Status, nilIfEmpty(w.ReviewStatus), nilIfEmpty(w.DeleteKind),
		w.DeletedAt, w.DeletedByID)
	if err != nil {
		return fmt.Errorf("CreateWork.write: %w", err)
	}
	return nil
}

// DeleteWork soft-deletes a work.
type DeleteWork struct {
	WorkID     ID     `json:"work_id"`
	DeleteKind string // withdrawn, retracted, takedown

	work *Work // populated by apply
}

func (m *DeleteWork) name() string { return "delete:work" }

func (m *DeleteWork) needs() updateNeeds {
	return updateNeeds{workIDs: []ID{m.WorkID}}
}

func (m *DeleteWork) apply(state updateState, userID *ID) (*updateEffect, error) {
	w, ok := state.works[m.WorkID]
	if !ok {
		return nil, fmt.Errorf("DeleteWork: work %s not found", m.WorkID)
	}
	if w.Status == WorkStatusDeleted {
		return nil, nil // noop
	}

	now := time.Now()
	w.Version++
	w.Status = WorkStatusDeleted
	w.DeleteKind = m.DeleteKind
	w.DeletedAt = &now
	m.work = w

	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		record:     w,
	}, nil
}

func (m *DeleteWork) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	w := m.work
	_, err := tx.Exec(ctx, `
		UPDATE bbl_works
		SET version = $2, updated_at = transaction_timestamp(),
		    updated_by_id = $3, status = $4, delete_kind = $5,
		    deleted_at = $6, deleted_by_id = $7
		WHERE id = $1`,
		w.ID, w.Version, w.UpdatedByID,
		w.Status, nilIfEmpty(w.DeleteKind),
		w.DeletedAt, w.DeletedByID)
	if err != nil {
		return fmt.Errorf("DeleteWork.write: %w", err)
	}
	return nil
}
