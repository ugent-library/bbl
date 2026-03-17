package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for organization relation tables ---

func writeOrganizationAssertion(ctx context.Context, tx pgx.Tx, id, organizationID ID, field string, val any, hidden bool, organizationSourceID *ID, userID *ID) error {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_assertions (id, organization_id, field, val, hidden, organization_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, organizationID, field, valJSON, hidden, organizationSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
	}
	return nil
}

func writeOrganizationName(ctx context.Context, tx pgx.Tx, id, assertionID, organizationID ID, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_names (id, assertion_id, organization_id, lang, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, organizationID, lang, val)
	if err != nil {
		return fmt.Errorf("writeOrganizationName: %w", err)
	}
	return nil
}

func writeOrganizationIdentifier(ctx context.Context, tx pgx.Tx, id, assertionID, organizationID ID, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_identifiers (id, assertion_id, organization_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, organizationID, scheme, val)
	if err != nil {
		return fmt.Errorf("writeOrganizationIdentifier: %w", err)
	}
	return nil
}

func writeOrganizationRel(ctx context.Context, tx pgx.Tx, id, assertionID, organizationID, relOrganizationID ID, kind string, startDate, endDate *time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_rels (id, assertion_id, organization_id, rel_organization_id, kind, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, assertionID, organizationID, relOrganizationID, kind, startDate, endDate)
	if err != nil {
		return fmt.Errorf("writeOrganizationRel: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset mutations for organization collectives
// ============================================================

// --- SetOrganizationNames (no delete — required) ---

type SetOrganizationNames struct {
	OrganizationID ID     `json:"organization_id"`
	Names          []Text
	userID         *ID
}

func (m *SetOrganizationNames) mutationName() string { return "set_organization_names" }
func (m *SetOrganizationNames) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationNames) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Names []Text }{m.Names}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "names", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationNames) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'names' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationNames: delete: %w", err)
	}
	assertionID := newID()
	if err := writeOrganizationAssertion(ctx, tx, assertionID, m.OrganizationID, "names", nil, false, nil, m.userID); err != nil {
		return fmt.Errorf("SetOrganizationNames: %w", err)
	}
	for _, t := range m.Names {
		if err := writeOrganizationName(ctx, tx, newID(), assertionID, m.OrganizationID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetOrganizationNames: %w", err)
		}
	}
	return nil
}

// --- SetOrganizationIdentifiers / UnsetOrganizationIdentifiers ---

type SetOrganizationIdentifiers struct {
	OrganizationID ID     `json:"organization_id"`
	Identifiers    []Identifier
	userID         *ID
}

func (m *SetOrganizationIdentifiers) mutationName() string { return "set_organization_identifiers" }
func (m *SetOrganizationIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Identifiers []Identifier }{m.Identifiers}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: delete: %w", err)
	}
	assertionID := newID()
	if err := writeOrganizationAssertion(ctx, tx, assertionID, m.OrganizationID, "identifiers", nil, false, nil, m.userID); err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeOrganizationIdentifier(ctx, tx, newID(), assertionID, m.OrganizationID, ident.Scheme, ident.Val); err != nil {
			return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationIdentifiers struct{ OrganizationID ID }

func (m *UnsetOrganizationIdentifiers) mutationName() string { return "unset_organization_identifiers" }
func (m *UnsetOrganizationIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetOrganizationIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("UnsetOrganizationIdentifiers: %w", err)
	}
	return nil
}

// --- SetOrganizationRels / UnsetOrganizationRels ---

type SetOrganizationRels struct {
	OrganizationID ID     `json:"organization_id"`
	Rels []struct {
		RelOrganizationID ID     `json:"rel_organization_id"`
		Kind              string `json:"kind"`
	} `json:"rels"`
	userID *ID
}

func (m *SetOrganizationRels) mutationName() string { return "set_organization_rels" }
func (m *SetOrganizationRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetOrganizationRels) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpUpdate,
		diff:       Diff{Args: m.Rels},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("SetOrganizationRels: delete: %w", err)
	}
	assertionID := newID()
	if err := writeOrganizationAssertion(ctx, tx, assertionID, m.OrganizationID, "rels", nil, false, nil, m.userID); err != nil {
		return fmt.Errorf("SetOrganizationRels: %w", err)
	}
	for _, r := range m.Rels {
		if err := writeOrganizationRel(ctx, tx, newID(), assertionID, m.OrganizationID, r.RelOrganizationID, r.Kind, nil, nil); err != nil {
			return fmt.Errorf("SetOrganizationRels: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationRels struct{ OrganizationID ID }

func (m *UnsetOrganizationRels) mutationName() string { return "unset_organization_rels" }
func (m *UnsetOrganizationRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetOrganizationRels) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("UnsetOrganizationRels: %w", err)
	}
	return nil
}
