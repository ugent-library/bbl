package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreatePersonField inserts a scalar assertion into bbl_person_assertions.
// Shared by both Set mutations (human path) and import.
func writeCreatePersonField(ctx context.Context, tx pgx.Tx, id, personID ID, field string, val any, personSourceID *ID, userID *ID) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_person_assertions (id, person_id, field, val, person_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, personID, field, valJSON, personSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Unset helpers for scalar fields ---

func applySetPersonField(personID ID, field string, val string, id *ID, mutUserID **ID, userID *ID) (*mutationEffect, error) {
	*id = newID()
	*mutUserID = userID
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		opType:     OpUpdate,
		diff:       Diff{Args: val},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeSetPersonField(ctx context.Context, tx pgx.Tx, id, personID ID, field string, val string, userID *ID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		personID, field); err != nil {
		return fmt.Errorf("writeSetPersonField(%s): delete: %w", field, err)
	}
	return writeCreatePersonField(ctx, tx, id, personID, field, val, nil, userID)
}

func applyUnsetPersonField(personID ID, field string) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeUnsetPersonField(ctx context.Context, tx pgx.Tx, personID ID, field string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		personID, field); err != nil {
		return fmt.Errorf("writeUnsetPersonField(%s): %w", field, err)
	}
	return nil
}

// --- shared write helpers for relation tables ---

func writePersonAssertion(ctx context.Context, tx pgx.Tx, id, personID ID, field string, val any, hidden bool, personSourceID *ID, userID *ID) error {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return fmt.Errorf("writePersonAssertion(%s): %w", field, err)
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_assertions (id, person_id, field, val, hidden, person_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, personID, field, valJSON, hidden, personSourceID, userID)
	if err != nil {
		return fmt.Errorf("writePersonAssertion(%s): %w", field, err)
	}
	return nil
}

func writePersonIdentifier(ctx context.Context, tx pgx.Tx, id, assertionID, personID ID, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_identifiers (id, assertion_id, person_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, personID, scheme, val)
	if err != nil {
		return fmt.Errorf("writePersonIdentifier: %w", err)
	}
	return nil
}

func writePersonOrganization(ctx context.Context, tx pgx.Tx, id, assertionID, personID, organizationID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_organizations (id, assertion_id, person_id, organization_id)
		VALUES ($1, $2, $3, $4)`,
		id, assertionID, personID, organizationID)
	if err != nil {
		return fmt.Errorf("writePersonOrganization: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset mutations for person scalar fields
// ============================================================

// --- SetPersonName (no delete — required) ---

type SetPersonName struct {
	PersonID ID     `json:"person_id"`
	Val      string
	id       ID
	userID   *ID
}

func (m *SetPersonName) mutationName() string { return "set_person_name" }
func (m *SetPersonName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetPersonField(m.PersonID, "name", m.Val, &m.id, &m.userID, userID)
}
func (m *SetPersonName) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetPersonField(ctx, tx, m.id, m.PersonID, "name", m.Val, m.userID)
}

// --- SetPersonGivenName / UnsetPersonGivenName ---

type SetPersonGivenName struct {
	PersonID ID     `json:"person_id"`
	Val      string
	id       ID
	userID   *ID
}

func (m *SetPersonGivenName) mutationName() string { return "set_person_given_name" }
func (m *SetPersonGivenName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonGivenName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetPersonField(m.PersonID, "given_name", m.Val, &m.id, &m.userID, userID)
}
func (m *SetPersonGivenName) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetPersonField(ctx, tx, m.id, m.PersonID, "given_name", m.Val, m.userID)
}

type UnsetPersonGivenName struct{ PersonID ID }

func (m *UnsetPersonGivenName) mutationName() string { return "unset_person_given_name" }
func (m *UnsetPersonGivenName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetPersonGivenName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetPersonField(m.PersonID, "given_name")
}
func (m *UnsetPersonGivenName) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetPersonField(ctx, tx, m.PersonID, "given_name")
}

// --- SetPersonMiddleName / UnsetPersonMiddleName ---

type SetPersonMiddleName struct {
	PersonID ID     `json:"person_id"`
	Val      string
	id       ID
	userID   *ID
}

func (m *SetPersonMiddleName) mutationName() string { return "set_person_middle_name" }
func (m *SetPersonMiddleName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonMiddleName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetPersonField(m.PersonID, "middle_name", m.Val, &m.id, &m.userID, userID)
}
func (m *SetPersonMiddleName) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetPersonField(ctx, tx, m.id, m.PersonID, "middle_name", m.Val, m.userID)
}

type UnsetPersonMiddleName struct{ PersonID ID }

func (m *UnsetPersonMiddleName) mutationName() string { return "unset_person_middle_name" }
func (m *UnsetPersonMiddleName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetPersonMiddleName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetPersonField(m.PersonID, "middle_name")
}
func (m *UnsetPersonMiddleName) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetPersonField(ctx, tx, m.PersonID, "middle_name")
}

// --- SetPersonFamilyName / UnsetPersonFamilyName ---

type SetPersonFamilyName struct {
	PersonID ID     `json:"person_id"`
	Val      string
	id       ID
	userID   *ID
}

func (m *SetPersonFamilyName) mutationName() string { return "set_person_family_name" }
func (m *SetPersonFamilyName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonFamilyName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetPersonField(m.PersonID, "family_name", m.Val, &m.id, &m.userID, userID)
}
func (m *SetPersonFamilyName) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetPersonField(ctx, tx, m.id, m.PersonID, "family_name", m.Val, m.userID)
}

type UnsetPersonFamilyName struct{ PersonID ID }

func (m *UnsetPersonFamilyName) mutationName() string { return "unset_person_family_name" }
func (m *UnsetPersonFamilyName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetPersonFamilyName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetPersonField(m.PersonID, "family_name")
}
func (m *UnsetPersonFamilyName) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetPersonField(ctx, tx, m.PersonID, "family_name")
}

// ============================================================
// Set / Unset mutations for person collectives
// ============================================================

// --- SetPersonIdentifiers / UnsetPersonIdentifiers ---

type SetPersonIdentifiers struct {
	PersonID    ID
	Identifiers []Identifier `json:"identifiers"`
	userID      *ID
}

func (m *SetPersonIdentifiers) mutationName() string { return "set_person_identifiers" }
func (m *SetPersonIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Identifiers []Identifier }{m.Identifiers}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "identifiers", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("SetPersonIdentifiers: delete: %w", err)
	}
	assertionID := newID()
	if err := writePersonAssertion(ctx, tx, assertionID, m.PersonID, "identifiers", nil, false, nil, m.userID); err != nil {
		return fmt.Errorf("SetPersonIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writePersonIdentifier(ctx, tx, newID(), assertionID, m.PersonID, ident.Scheme, ident.Val); err != nil {
			return fmt.Errorf("SetPersonIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetPersonIdentifiers struct{ PersonID ID }

func (m *UnsetPersonIdentifiers) mutationName() string { return "unset_person_identifiers" }
func (m *UnsetPersonIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetPersonIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "identifiers", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *UnsetPersonIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("UnsetPersonIdentifiers: %w", err)
	}
	return nil
}

// --- SetPersonOrganizations / UnsetPersonOrganizations ---

type SetPersonOrganizations struct {
	PersonID      ID
	Organizations []PersonOrganization `json:"organizations"`
	userID        *ID
}

func (m *SetPersonOrganizations) mutationName() string { return "set_person_organizations" }
func (m *SetPersonOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetPersonOrganizations) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Organizations []PersonOrganization }{m.Organizations}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "organizations", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("SetPersonOrganizations: delete: %w", err)
	}
	assertionID := newID()
	if err := writePersonAssertion(ctx, tx, assertionID, m.PersonID, "organizations", nil, false, nil, m.userID); err != nil {
		return fmt.Errorf("SetPersonOrganizations: %w", err)
	}
	for _, org := range m.Organizations {
		if err := writePersonOrganization(ctx, tx, newID(), assertionID, m.PersonID, org.OrganizationID); err != nil {
			return fmt.Errorf("SetPersonOrganizations: %w", err)
		}
	}
	return nil
}

type UnsetPersonOrganizations struct{ PersonID ID }

func (m *UnsetPersonOrganizations) mutationName() string { return "unset_person_organizations" }
func (m *UnsetPersonOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetPersonOrganizations) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "organizations", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *UnsetPersonOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("UnsetPersonOrganizations: %w", err)
	}
	return nil
}
