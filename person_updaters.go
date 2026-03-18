package bbl

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreatePerson creates a new person entity.
type CreatePerson struct {
	PersonID ID `json:"person_id"`

	person *Person // populated by apply
}

func (m *CreatePerson) name() string { return "create:person" }

func (m *CreatePerson) needs() updateNeeds { return updateNeeds{} }

func (m *CreatePerson) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.person = &Person{
		ID:      m.PersonID,
		Version: 1,
		Status:  PersonStatusPublic,
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		record:     m.person,
	}, nil
}

func (m *CreatePerson) write(ctx context.Context, tx pgx.Tx, revID int64) error {
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
	PersonID ID `json:"person_id"`

	person *Person // populated by apply
}

func (m *DeletePerson) name() string { return "delete:person" }

func (m *DeletePerson) needs() updateNeeds {
	return updateNeeds{personIDs: []ID{m.PersonID}}
}

func (m *DeletePerson) apply(state updateState, userID *ID) (*updateEffect, error) {
	p, ok := state.people[m.PersonID]
	if !ok {
		return nil, fmt.Errorf("DeletePerson: person %s not found", m.PersonID)
	}
	if p.Status == PersonStatusDeleted {
		return nil, nil // noop
	}

	now := time.Now()
	p.Version++
	p.Status = PersonStatusDeleted
	p.DeletedAt = &now
	m.person = p

	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		record:     p,
	}, nil
}

func (m *DeletePerson) write(ctx context.Context, tx pgx.Tx, revID int64) error {
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
