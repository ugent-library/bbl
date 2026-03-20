package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for organization assertion tables ---

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

func writeOrganizationRel(ctx context.Context, tx pgx.Tx, assertionID int64, relOrganizationID ID, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_organization_assertion_rels (assertion_id, rel_organization_id, kind)
		VALUES ($1, $2, $3)`,
		assertionID, relOrganizationID, kind)
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
	role           *string
}

func (m *SetOrganizationNames) name() string       { return "set:organization_names" }
func (m *SetOrganizationNames) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *SetOrganizationNames) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if o := state.organizations[m.OrganizationID]; o != nil && slicesEqual(o.Names, m.Names) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "names") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "names", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationNames) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "names", revID); err != nil {
		return fmt.Errorf("SetOrganizationNames: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.OrganizationID, "names"); err != nil {
		return fmt.Errorf("SetOrganizationNames: %w", err)
	}
	for _, t := range m.Names {
		if _, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "names", t, false, nil, m.userID, m.role); err != nil {
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
	role           *string
}

func (m *SetOrganizationIdentifiers) name() string       { return "set:organization_identifiers" }
func (m *SetOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *SetOrganizationIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if o := state.organizations[m.OrganizationID]; o != nil && slicesEqual(o.Identifiers, m.Identifiers) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "identifiers", revID); err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.OrganizationID, "identifiers"); err != nil {
		return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if _, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "identifiers", ident, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetOrganizationIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationIdentifiers struct{ OrganizationID ID }

func (m *UnsetOrganizationIdentifiers) name() string       { return "unset:organization_identifiers" }
func (m *UnsetOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *UnsetOrganizationIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if o := state.organizations[m.OrganizationID]; o != nil && len(o.Identifiers) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "identifiers", revID); err != nil {
		return fmt.Errorf("UnsetOrganizationIdentifiers: %w", err)
	}
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
	role   *string
}

func (m *SetOrganizationRels) name() string       { return "set:organization_rels" }
func (m *SetOrganizationRels) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *SetOrganizationRels) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if o := state.organizations[m.OrganizationID]; o != nil && orgRelsMatch(o.Rels, m.Rels) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "rels") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *SetOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "rels", revID); err != nil {
		return fmt.Errorf("SetOrganizationRels: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.OrganizationID, "rels"); err != nil {
		return fmt.Errorf("SetOrganizationRels: %w", err)
	}
	for _, r := range m.Rels {
		val := struct {
			Kind string `json:"kind"`
		}{r.Kind}
		assertionID, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "rels", val, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetOrganizationRels: %w", err)
		}
		if err := writeOrganizationRel(ctx, tx, assertionID, r.RelOrganizationID, r.Kind); err != nil {
			return fmt.Errorf("SetOrganizationRels: %w", err)
		}
	}
	return nil
}

type UnsetOrganizationRels struct{ OrganizationID ID }

func (m *UnsetOrganizationRels) name() string       { return "unset:organization_rels" }
func (m *UnsetOrganizationRels) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *UnsetOrganizationRels) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if o := state.organizations[m.OrganizationID]; o != nil && len(o.Rels) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "rels") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *UnsetOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "rels", revID); err != nil {
		return fmt.Errorf("UnsetOrganizationRels: %w", err)
	}
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
	role           *string
}

func (m *HideOrganizationIdentifiers) name() string       { return "hide:organization_identifiers" }
func (m *HideOrganizationIdentifiers) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *HideOrganizationIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.organizationAssertions[m.OrganizationID], "identifiers") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "identifiers", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *HideOrganizationIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "identifiers", revID); err != nil {
		return fmt.Errorf("HideOrganizationIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.OrganizationID, "identifiers"); err != nil {
		return fmt.Errorf("HideOrganizationIdentifiers: %w", err)
	}
	_, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "identifiers", nil, true, nil, m.userID, m.role)
	return err
}

// --- HideOrganizationRels ---

type HideOrganizationRels struct {
	OrganizationID ID
	userID         *ID
	role           *string
}

func (m *HideOrganizationRels) name() string       { return "hide:organization_rels" }
func (m *HideOrganizationRels) needs() updateNeeds { return updateNeeds{organizationIDs: []ID{m.OrganizationID}} }
func (m *HideOrganizationRels) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.organizationAssertions[m.OrganizationID], "rels") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.organizationAssertions[m.OrganizationID], "rels") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypeOrganization,
		recordID:   m.OrganizationID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_organization_assertions", "organization_id", m.OrganizationID, "rels", "organization_source_id", "bbl_organization_sources", priorities)
		},
	}, nil
}
func (m *HideOrganizationRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logOrganizationHistory(ctx, tx, m.OrganizationID, "rels", revID); err != nil {
		return fmt.Errorf("HideOrganizationRels: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_organization_assertions WHERE organization_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.OrganizationID, "rels"); err != nil {
		return fmt.Errorf("HideOrganizationRels: %w", err)
	}
	_, err := writeOrganizationAssertion(ctx, tx, revID, m.OrganizationID, "rels", nil, true, nil, m.userID, m.role)
	return err
}
