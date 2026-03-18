package bbl

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateOrganization creates a new organization entity.
type CreateOrganization struct {
	OrganizationID ID
	Kind           string
	StartDate      *time.Time
	EndDate        *time.Time

	org *Organization // populated by apply
}

func (m *CreateOrganization) name() string { return "create:organization" }

func (m *CreateOrganization) needs() updateNeeds { return updateNeeds{} }

func (m *CreateOrganization) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.org = &Organization{
		ID:        m.OrganizationID,
		Version:   1,
		Kind:      m.Kind,
		Status:    OrganizationStatusPublic,
		StartDate: m.StartDate,
		EndDate:   m.EndDate,
	}
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		record:     m.org,
	}, nil
}

func (m *CreateOrganization) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	o := m.org
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organizations
		    (id, version, created_by_id, updated_by_id,
		     kind, status, start_date, end_date,
		     deleted_at, deleted_by_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		o.ID, o.Version, o.CreatedByID, o.UpdatedByID,
		o.Kind, o.Status, o.StartDate, o.EndDate,
		o.DeletedAt, o.DeletedByID)
	if err != nil {
		return fmt.Errorf("CreateOrganization.write: %w", err)
	}
	return nil
}

// DeleteOrganization soft-deletes an organization.
type DeleteOrganization struct {
	OrganizationID ID `json:"organization_id"`

	org *Organization // populated by apply
}

func (m *DeleteOrganization) name() string { return "delete:organization" }

func (m *DeleteOrganization) needs() updateNeeds {
	return updateNeeds{organizationIDs: []ID{m.OrganizationID}}
}

func (m *DeleteOrganization) apply(state updateState, userID *ID) (*updateEffect, error) {
	o, ok := state.organizations[m.OrganizationID]
	if !ok {
		return nil, fmt.Errorf("DeleteOrganization: organization %s not found", m.OrganizationID)
	}
	if o.Status == OrganizationStatusDeleted {
		return nil, nil // noop
	}

	now := time.Now()
	o.Version++
	o.Status = OrganizationStatusDeleted
	o.DeletedAt = &now
	m.org = o

	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		record:     o,
	}, nil
}

func (m *DeleteOrganization) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	o := m.org
	_, err := tx.Exec(ctx, `
		UPDATE bbl_organizations
		SET version = $2, updated_at = transaction_timestamp(),
		    updated_by_id = $3, status = $4,
		    deleted_at = $5, deleted_by_id = $6
		WHERE id = $1`,
		o.ID, o.Version, o.UpdatedByID,
		o.Status, o.DeletedAt, o.DeletedByID)
	if err != nil {
		return fmt.Errorf("DeleteOrganization.write: %w", err)
	}
	return nil
}
