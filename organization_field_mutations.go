package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for organization relation tables ---
// These are used by both Set mutations (human path) and import.

func writeOrganizationName(ctx context.Context, tx pgx.Tx, id, organizationID ID, lang, val string, organizationSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_names (id, organization_id, lang, val, organization_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, organizationID, lang, val, organizationSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeOrganizationName: %w", err)
	}
	return nil
}

func writeOrganizationIdentifier(ctx context.Context, tx pgx.Tx, id, organizationID ID, scheme, val string, organizationSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_identifiers (id, organization_id, scheme, val, organization_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, organizationID, scheme, val, organizationSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeOrganizationIdentifier: %w", err)
	}
	return nil
}

func writeOrganizationRel(ctx context.Context, tx pgx.Tx, id, organizationID, relOrganizationID ID, kind string, organizationSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_rels (id, organization_id, rel_organization_id, kind, organization_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, organizationID, relOrganizationID, kind, organizationSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeOrganizationRel: %w", err)
	}
	return nil
}

// ============================================================
// Set / Delete mutations for organization collectives
// ============================================================

// --- SetOrganizationNames (no delete — required) ---

type SetOrganizationNames struct {
	OrganizationID ID
	Names          []Text
	userID         *ID
}

func (m *SetOrganizationNames) mutationName() string { return "SetOrganizationNames" }
func (m *SetOrganizationNames) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationNames) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Names []Text }{m.Names}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_organization_names", "organization_id", m.OrganizationID, "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationNames) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_names WHERE organization_id = $1 AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationNames: delete: %w", err)
	}
	for _, t := range m.Names {
		if err := writeOrganizationName(ctx, tx, newID(), m.OrganizationID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetOrganizationNames: %w", err)
		}
	}
	return nil
}

// --- SetOrganizationIdentifiers / DeleteOrganizationIdentifiers ---

type SetOrganizationIdentifiers struct {
	OrganizationID ID
	Identifiers    []Identifier
	userID         *ID
}

func (m *SetOrganizationIdentifiers) mutationName() string { return "SetOrganizationIdentifiers" }
func (m *SetOrganizationIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Identifiers []Identifier }{m.Identifiers}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_organization_identifiers", "organization_id", m.OrganizationID, "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_identifiers WHERE organization_id = $1 AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: delete: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeOrganizationIdentifier(ctx, tx, newID(), m.OrganizationID, ident.Scheme, ident.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
		}
	}
	return nil
}

type DeleteOrganizationIdentifiers struct{ OrganizationID ID }

func (m *DeleteOrganizationIdentifiers) mutationName() string { return "DeleteOrganizationIdentifiers" }
func (m *DeleteOrganizationIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteOrganizationIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_organization_identifiers", "organization_id", m.OrganizationID, "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *DeleteOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_identifiers WHERE organization_id = $1 AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("DeleteOrganizationIdentifiers: %w", err)
	}
	return nil
}

// --- SetOrganizationRels / DeleteOrganizationRels ---

type SetOrganizationRels struct {
	OrganizationID ID
	Rels           []struct {
		RelOrganizationID ID
		Kind              string
	}
	userID *ID
}

func (m *SetOrganizationRels) mutationName() string { return "SetOrganizationRels" }
func (m *SetOrganizationRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationRels) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: m.Rels},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_organization_rels", "organization_id", m.OrganizationID, "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_rels WHERE organization_id = $1 AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationRels: delete: %w", err)
	}
	for _, r := range m.Rels {
		if err := writeOrganizationRel(ctx, tx, newID(), m.OrganizationID, r.RelOrganizationID, r.Kind, nil, m.userID); err != nil {
			return fmt.Errorf("SetOrganizationRels: %w", err)
		}
	}
	return nil
}

type DeleteOrganizationRels struct{ OrganizationID ID }

func (m *DeleteOrganizationRels) mutationName() string { return "DeleteOrganizationRels" }
func (m *DeleteOrganizationRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteOrganizationRels) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_organization_rels", "organization_id", m.OrganizationID, "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *DeleteOrganizationRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_rels WHERE organization_id = $1 AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("DeleteOrganizationRels: %w", err)
	}
	return nil
}
