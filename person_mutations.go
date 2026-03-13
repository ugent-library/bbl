package bbl

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreatePerson creates a new person entity.
type CreatePerson struct {
	EntityID ID

	person *Person // populated by apply
}

func (m *CreatePerson) mutationName() string { return "CreatePerson" }

func (m *CreatePerson) needs() mutationNeeds { return mutationNeeds{} }

func (m *CreatePerson) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.person = &Person{
		ID:      m.EntityID,
		Version: 1,
		Status:  PersonStatusPublic,
	}
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.EntityID,
		opType:     OpCreate,
		diff:       Diff{Args: struct{}{}},
		record:     m.person,
	}, nil
}

func (m *CreatePerson) write(ctx context.Context, tx pgx.Tx) error {
	p := m.person
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_people
		    (id, version, created_by_id, updated_by_id,
		     status, deleted_at, deleted_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.Version, p.CreatedByID, p.UpdatedByID,
		p.Status, p.DeletedAt, p.DeletedByID)
	if err != nil {
		return fmt.Errorf("CreatePerson.write: %w", err)
	}
	return nil
}

// DeletePerson soft-deletes a person.
type DeletePerson struct {
	EntityID ID

	person *Person // populated by apply
}

func (m *DeletePerson) mutationName() string { return "DeletePerson" }

func (m *DeletePerson) needs() mutationNeeds {
	return mutationNeeds{personIDs: []ID{m.EntityID}}
}

func (m *DeletePerson) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	p, ok := state.people[m.EntityID]
	if !ok {
		return nil, fmt.Errorf("DeletePerson: person %s not found", m.EntityID)
	}
	if p.Status == PersonStatusDeleted {
		return nil, nil // noop
	}

	now := time.Now()
	p.Version++
	p.Status = PersonStatusDeleted
	p.DeletedAt = &now
	m.person = p

	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.EntityID,
		opType:     OpDelete,
		diff: Diff{
			Args: struct{}{},
			Prev: struct{ Status string }{p.Status},
		},
		record: p,
	}, nil
}

func (m *DeletePerson) write(ctx context.Context, tx pgx.Tx) error {
	p := m.person
	_, err := tx.Exec(ctx, `
		UPDATE bbl_people
		SET version = $2, updated_at = transaction_timestamp(),
		    updated_by_id = $3, status = $4,
		    deleted_at = $5, deleted_by_id = $6
		WHERE id = $1`,
		p.ID, p.Version, p.UpdatedByID,
		p.Status, p.DeletedAt, p.DeletedByID)
	if err != nil {
		return fmt.Errorf("DeletePerson.write: %w", err)
	}
	return nil
}
