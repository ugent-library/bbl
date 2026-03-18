package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for organization relation tables ---

func writeOrganizationAssertion(ctx context.Context, tx pgx.Tx, revID int64, organizationID ID, field string, val any, hidden bool, organizationSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_organization_assertions (rev_id, organization_id, field, val, hidden, organization_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, organizationID, field, valJSON, hidden, organizationSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeOrganizationAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeOrganizationName(ctx context.Context, tx pgx.Tx, id, organizationID ID, assertionID int64, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_names (id, assertion_id, organization_id, lang, val)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'und'), $5)`,
		id, assertionID, organizationID, lang, val)
	if err != nil {
		return fmt.Errorf("writeOrganizationName: %w", err)
	}
	return nil
}

func writeOrganizationIdentifier(ctx context.Context, tx pgx.Tx, id, organizationID ID, assertionID int64, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_identifiers (id, assertion_id, organization_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, organizationID, scheme, val)
	if err != nil {
		return fmt.Errorf("writeOrganizationIdentifier: %w", err)
	}
	return nil
}

func writeOrganizationRel(ctx context.Context, tx pgx.Tx, id, organizationID, relOrganizationID ID, assertionID int64, kind string, startDate, endDate *time.Time) error {
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
// Set / Unset updaters for organization collectives
// ============================================================

// --- SetOrganizationNames (no delete — required) ---

type SetOrganizationNames struct {
	OrganizationID ID `json:"organization_id"`
	Names          []Text
	userID         *ID
}

func (m *SetOrganizationNames) name() string       { return "set:organization_names" }
func (m *SetOrganizationNames) needs() updateNeeds { return updateNeeds{} }
func (m *SetOrganizationNames) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "names", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationNames) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "names", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetOrganizationNames: %w", err)
	}
	for _, t := range m.Names {
		if err := writeOrganizationName(ctx, tx, newID(), m.OrganizationID, assertionID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetOrganizationNames: %w", err)
		}
	}
	return nil
}

// --- SetOrganizationIdentifiers / UnsetOrganizationIdentifiers ---

type SetOrganizationIdentifiers struct {
	OrganizationID ID `json:"organization_id"`
	Identifiers    []Identifier
	userID         *ID
}

func (m *SetOrganizationIdentifiers) name() string       { return "set:organization_identifiers" }
func (m *SetOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *SetOrganizationIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "identifiers", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeOrganizationIdentifier(ctx, tx, newID(), m.OrganizationID, assertionID, ident.Scheme, ident.Val); err != nil {
			return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationIdentifiers struct{ OrganizationID ID }

func (m *UnsetOrganizationIdentifiers) name() string       { return "unset:organization_identifiers" }
func (m *UnsetOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *UnsetOrganizationIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("UnsetOrganizationIdentifiers: %w", err)
	}
	return nil
}

// --- SetOrganizationRels / UnsetOrganizationRels ---

type SetOrganizationRels struct {
	OrganizationID ID `json:"organization_id"`
	Rels           []struct {
		RelOrganizationID ID     `json:"rel_organization_id"`
		Kind              string `json:"kind"`
	} `json:"rels"`
	userID *ID
}

func (m *SetOrganizationRels) name() string       { return "set:organization_rels" }
func (m *SetOrganizationRels) needs() updateNeeds { return updateNeeds{} }
func (m *SetOrganizationRels) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "rels", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetOrganizationRels: %w", err)
	}
	for _, r := range m.Rels {
		if err := writeOrganizationRel(ctx, tx, newID(), m.OrganizationID, r.RelOrganizationID, assertionID, r.Kind, nil, nil); err != nil {
			return fmt.Errorf("SetOrganizationRels: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationRels struct{ OrganizationID ID }

func (m *UnsetOrganizationRels) name() string       { return "unset:organization_rels" }
func (m *UnsetOrganizationRels) needs() updateNeeds { return updateNeeds{} }
func (m *UnsetOrganizationRels) apply(state updateState, userID *ID) (*updateEffect, error) {
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.OrganizationID); err != nil {
		return fmt.Errorf("UnsetOrganizationRels: %w", err)
	}
	return nil
}

// ============================================================
// Hide updaters for organization fields
// ============================================================

// --- HideOrganizationIdentifiers ---

type HideOrganizationIdentifiers struct {
	OrganizationID ID
	userID         *ID
}

func (m *HideOrganizationIdentifiers) name() string       { return "hide:organization_identifiers" }
func (m *HideOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{} }
func (m *HideOrganizationIdentifiers) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *HideOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	_, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "identifiers", nil, true, nil, m.userID, nil)
	return err
}

// --- HideOrganizationRels ---

type HideOrganizationRels struct {
	OrganizationID ID
	userID         *ID
}

func (m *HideOrganizationRels) name() string       { return "hide:organization_rels" }
func (m *HideOrganizationRels) needs() updateNeeds { return updateNeeds{} }
func (m *HideOrganizationRels) apply(state updateState, userID *ID) (*updateEffect, error) {
	m.userID = userID
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *HideOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	_, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "rels", nil, true, nil, m.userID, nil)
	return err
}
