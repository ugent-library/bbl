package bbl

import "fmt"

// CreatePerson creates a new person entity.
type CreatePerson struct {
	PersonID ID `json:"person_id"`
}

func (m *CreatePerson) name() string { return "create:person" }

func (m *CreatePerson) needs() updateNeeds { return updateNeeds{} }

func (m *CreatePerson) apply(state updateState, user *User) (*updateEffect, error) {
	state.records[m.PersonID] = &recordState{
		recordType: RecordTypePerson,
		id:         m.PersonID,
		status:     PersonStatusPublic,
		fields:     make(map[string]any),
		assertions: make(map[string]*fieldState),
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
	}, nil
}

func (m *CreatePerson) write(revID int64, user *User) (string, []any) {
	return `INSERT INTO bbl_people
		    (id, version, created_by_id, updated_by_id, status)
		VALUES ($1, 1, $2, $3, $4)`,
		[]any{m.PersonID, &user.ID, &user.ID, PersonStatusPublic}
}

// DeletePerson soft-deletes a person.
type DeletePerson struct {
	PersonID ID `json:"person_id"`
}

func (m *DeletePerson) name() string { return "delete:person" }

func (m *DeletePerson) needs() updateNeeds {
	return updateNeeds{personIDs: []ID{m.PersonID}}
}

func (m *DeletePerson) apply(state updateState, user *User) (*updateEffect, error) {
	rs := state.records[m.PersonID]
	if rs == nil {
		return nil, fmt.Errorf("DeletePerson: person %s not found", m.PersonID)
	}
	if rs.status == PersonStatusDeleted {
		return nil, nil // noop
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
	}, nil
}

func (m *DeletePerson) write(revID int64, user *User) (string, []any) {
	return `UPDATE bbl_people
		SET status = $2,
		    deleted_at = transaction_timestamp(), deleted_by_id = $3
		WHERE id = $1`,
		[]any{m.PersonID, PersonStatusDeleted, &user.ID}
}
