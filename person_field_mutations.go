package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreatePersonField inserts a scalar assertion into bbl_person_fields.
// Shared by both Set mutations (human path) and import.
func writeCreatePersonField(ctx context.Context, tx pgx.Tx, id, personID ID, field string, val any, personSourceID *ID, userID *ID) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_person_fields (id, person_id, field, val, person_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, personID, field, valJSON, personSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Delete helpers for scalar fields ---

func applySetPersonField(personID ID, field string, val string, id *ID, mutUserID **ID, userID *ID) (*mutationEffect, error) {
	*id = newID()
	*mutUserID = userID
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		opType:     OpUpdate,
		diff:       Diff{Args: val},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinScalar(ctx, tx, "bbl_person_fields", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeSetPersonField(ctx context.Context, tx pgx.Tx, id, personID ID, field string, val string, userID *ID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_person_fields WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		personID, field); err != nil {
		return fmt.Errorf("writeSetPersonField(%s): delete: %w", field, err)
	}
	return writeCreatePersonField(ctx, tx, id, personID, field, val, nil, userID)
}

func applyDeletePersonField(personID ID, field string) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinScalar(ctx, tx, "bbl_person_fields", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeDeletePersonField(ctx context.Context, tx pgx.Tx, personID ID, field string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_person_fields WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		personID, field); err != nil {
		return fmt.Errorf("writeDeletePersonField(%s): %w", field, err)
	}
	return nil
}

// --- shared write helpers for relation tables ---

func writePersonIdentifier(ctx context.Context, tx pgx.Tx, id, personID ID, scheme, val string, personSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_identifiers (id, person_id, scheme, val, person_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, personID, scheme, val, personSourceID, userID)
	if err != nil {
		return fmt.Errorf("writePersonIdentifier: %w", err)
	}
	return nil
}

func writePersonOrganization(ctx context.Context, tx pgx.Tx, id, personID, organizationID ID, role string, personSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_organizations (id, person_id, organization_id, role, person_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, personID, organizationID, nilIfEmpty(role), personSourceID, userID)
	if err != nil {
		return fmt.Errorf("writePersonOrganization: %w", err)
	}
	return nil
}

// ============================================================
// Set / Delete mutations for person scalar fields
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

// --- SetPersonGivenName / DeletePersonGivenName ---

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

type DeletePersonGivenName struct{ PersonID ID }

func (m *DeletePersonGivenName) mutationName() string { return "delete_person_given_name" }
func (m *DeletePersonGivenName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeletePersonGivenName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeletePersonField(m.PersonID, "given_name")
}
func (m *DeletePersonGivenName) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeletePersonField(ctx, tx, m.PersonID, "given_name")
}

// --- SetPersonMiddleName / DeletePersonMiddleName ---

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

type DeletePersonMiddleName struct{ PersonID ID }

func (m *DeletePersonMiddleName) mutationName() string { return "delete_person_middle_name" }
func (m *DeletePersonMiddleName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeletePersonMiddleName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeletePersonField(m.PersonID, "middle_name")
}
func (m *DeletePersonMiddleName) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeletePersonField(ctx, tx, m.PersonID, "middle_name")
}

// --- SetPersonFamilyName / DeletePersonFamilyName ---

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

type DeletePersonFamilyName struct{ PersonID ID }

func (m *DeletePersonFamilyName) mutationName() string { return "delete_person_family_name" }
func (m *DeletePersonFamilyName) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeletePersonFamilyName) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeletePersonField(m.PersonID, "family_name")
}
func (m *DeletePersonFamilyName) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeletePersonField(ctx, tx, m.PersonID, "family_name")
}

// ============================================================
// Set / Delete mutations for person collectives
// ============================================================

// --- SetPersonIdentifiers / DeletePersonIdentifiers ---

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
			return autoPinCollective(ctx, tx, "bbl_person_identifiers", "person_id", m.PersonID, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_identifiers WHERE person_id = $1 AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("SetPersonIdentifiers: delete: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writePersonIdentifier(ctx, tx, newID(), m.PersonID, ident.Scheme, ident.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetPersonIdentifiers: %w", err)
		}
	}
	return nil
}

type DeletePersonIdentifiers struct{ PersonID ID }

func (m *DeletePersonIdentifiers) mutationName() string { return "delete_person_identifiers" }
func (m *DeletePersonIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeletePersonIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_person_identifiers", "person_id", m.PersonID, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *DeletePersonIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_identifiers WHERE person_id = $1 AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("DeletePersonIdentifiers: %w", err)
	}
	return nil
}

// --- SetPersonOrganizations / DeletePersonOrganizations ---

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
			return autoPinCollective(ctx, tx, "bbl_person_organizations", "person_id", m.PersonID, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_organizations WHERE person_id = $1 AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("SetPersonOrganizations: delete: %w", err)
	}
	for _, org := range m.Organizations {
		if err := writePersonOrganization(ctx, tx, newID(), m.PersonID, org.OrganizationID, org.Role, nil, m.userID); err != nil {
			return fmt.Errorf("SetPersonOrganizations: %w", err)
		}
	}
	return nil
}

type DeletePersonOrganizations struct{ PersonID ID }

func (m *DeletePersonOrganizations) mutationName() string { return "delete_person_organizations" }
func (m *DeletePersonOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeletePersonOrganizations) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_person_organizations", "person_id", m.PersonID, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *DeletePersonOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_organizations WHERE person_id = $1 AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("DeletePersonOrganizations: %w", err)
	}
	return nil
}
