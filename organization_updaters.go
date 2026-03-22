package bbl

import (
	"fmt"
	"time"
)

// CreateOrganization creates a new organization entity.
type CreateOrganization struct {
	ID        ID `json:"id"`
	Kind      string
	StartDate *time.Time
	EndDate   *time.Time
}

func (m *CreateOrganization) name() string { return "create:organization" }

func (m *CreateOrganization) needs() updateNeeds { return updateNeeds{} }

func (m *CreateOrganization) apply(state updateState, user *User) (*updateEffect, error) {
	state.records[m.ID] = &recordState{
		recordType: RecordTypeOrganization,
		id:         m.ID,
		status:     OrganizationStatusPublic,
		kind:       m.Kind,
		fields:     make(map[string]any),
		assertions: make(map[string][]assertion),
	}
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.ID,
	}, nil
}

func (m *CreateOrganization) write(revID int64, user *User) (string, []any) {
	return `INSERT INTO bbl_organizations
		    (id, version, created_by_id, updated_by_id,
		     kind, status, start_date, end_date)
		VALUES ($1, 1, $2, $3, $4, $5, $6, $7)`,
		[]any{m.ID, &user.ID, &user.ID,
			m.Kind, OrganizationStatusPublic, m.StartDate, m.EndDate}
}

// DeleteOrganization soft-deletes an organization.
type DeleteOrganization struct {
	OrganizationID ID `json:"id"`
}

func (m *DeleteOrganization) name() string { return "delete:organization" }

func (m *DeleteOrganization) needs() updateNeeds {
	return updateNeeds{organizationIDs: []ID{m.OrganizationID}}
}

func (m *DeleteOrganization) apply(state updateState, user *User) (*updateEffect, error) {
	rs := state.records[m.OrganizationID]
	if rs == nil {
		return nil, fmt.Errorf("DeleteOrganization: organization %s not found", m.OrganizationID)
	}
	if rs.status == OrganizationStatusDeleted {
		return nil, nil // noop
	}
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
	}, nil
}

func (m *DeleteOrganization) write(revID int64, user *User) (string, []any) {
	return `UPDATE bbl_organizations
		SET status = $2,
		    deleted_at = transaction_timestamp(), deleted_by_id = $3
		WHERE id = $1`,
		[]any{m.OrganizationID, OrganizationStatusDeleted, &user.ID}
}
