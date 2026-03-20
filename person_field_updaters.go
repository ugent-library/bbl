package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreatePersonField inserts a scalar assertion into bbl_person_assertions.
// Shared by both Set updaters (human path) and import.
func writeCreatePersonField(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, val any, personSourceID *ID, userID *ID, role *string) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_person_assertions (rev_id, person_id, field, val, person_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		revID, personID, field, valJSON, personSourceID, userID, role)
	if err != nil {
		return fmt.Errorf("writeCreatePersonField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Unset helpers for scalar fields ---

func applySetPersonField(state updateState, personID ID, field string, mutUserID **ID, mutRole **string, userID *ID, role string) (*updateEffect, error) {
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[personID], field) {
			return nil, ErrCuratorLock
		}
	}
	*mutUserID = userID
	*mutRole = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeSetPersonField(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, val string, userID *ID, role *string) error {
	if err := logPersonHistory(ctx, tx, personID, field, revID); err != nil {
		return fmt.Errorf("writeSetPersonField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, personID, field); err != nil {
		return fmt.Errorf("writeSetPersonField(%s): %w", field, err)
	}
	return writeCreatePersonField(ctx, tx, revID, personID, field, val, nil, userID, role)
}

func applyUnsetPersonField(state updateState, role string, personID ID, field string) (*updateEffect, error) {
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[personID], field) {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeUnsetPersonField(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string) error {
	if err := logPersonHistory(ctx, tx, personID, field, revID); err != nil {
		return fmt.Errorf("writeUnsetPersonField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		personID, field); err != nil {
		return fmt.Errorf("writeUnsetPersonField(%s): %w", field, err)
	}
	return nil
}

// --- Hide helpers for scalar fields ---

func applyHidePersonField(state updateState, personID ID, field string, mutUserID **ID, mutRole **string, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.personAssertions[personID], field) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[personID], field) {
			return nil, ErrCuratorLock
		}
	}
	*mutUserID = userID
	*mutRole = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   personID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", personID, field, "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}

func writeHidePersonField(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, userID *ID, role *string) error {
	if err := logPersonHistory(ctx, tx, personID, field, revID); err != nil {
		return fmt.Errorf("writeHidePersonField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, personID, field); err != nil {
		return fmt.Errorf("writeHidePersonField(%s): %w", field, err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_assertions (rev_id, person_id, field, val, hidden, person_source_id, user_id, role)
		VALUES ($1, $2, $3, NULL, true, NULL, $4, $5)`,
		revID, personID, field, userID, role)
	if err != nil {
		return fmt.Errorf("writeHidePersonField(%s): %w", field, err)
	}
	return nil
}

// --- shared write helpers for relation tables ---

func writePersonAssertion(ctx context.Context, tx pgx.Tx, revID int64, personID ID, field string, val any, hidden bool, personSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writePersonAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_person_assertions (rev_id, person_id, field, val, hidden, person_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, personID, field, valJSON, hidden, personSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writePersonAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writePersonOrganization(ctx context.Context, tx pgx.Tx, assertionID int64, orgID ID, validFrom, validTo *time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_person_assertion_organizations (assertion_id, organization_id, valid_from, valid_to)
		VALUES ($1, $2, $3, $4)`,
		assertionID, orgID, validFrom, validTo)
	if err != nil {
		return fmt.Errorf("writePersonOrganization: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset updaters for person scalar fields
// ============================================================

// --- SetPersonName (no delete — required) ---

type SetPersonName struct {
	PersonID ID `json:"person_id"`
	Val      string
	userID   *ID
	role     *string
}

func (m *SetPersonName) name() string       { return "set:person_name" }
func (m *SetPersonName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.Name == m.Val {
		return nil, nil
	}
	return applySetPersonField(state, m.PersonID, "name", &m.userID, &m.role, userID, role)
}
func (m *SetPersonName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetPersonField(ctx, tx, revID, m.PersonID, "name", m.Val, m.userID, m.role)
}

// --- SetPersonGivenName / UnsetPersonGivenName ---

type SetPersonGivenName struct {
	PersonID ID `json:"person_id"`
	Val      string
	userID   *ID
	role     *string
}

func (m *SetPersonGivenName) name() string       { return "set:person_given_name" }
func (m *SetPersonGivenName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonGivenName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.GivenName == m.Val {
		return nil, nil
	}
	return applySetPersonField(state, m.PersonID, "given_name", &m.userID, &m.role, userID, role)
}
func (m *SetPersonGivenName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetPersonField(ctx, tx, revID, m.PersonID, "given_name", m.Val, m.userID, m.role)
}

type UnsetPersonGivenName struct{ PersonID ID }

func (m *UnsetPersonGivenName) name() string       { return "unset:person_given_name" }
func (m *UnsetPersonGivenName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *UnsetPersonGivenName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.GivenName == "" {
		return nil, nil
	}
	return applyUnsetPersonField(state, role, m.PersonID, "given_name")
}
func (m *UnsetPersonGivenName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetPersonField(ctx, tx, revID, m.PersonID, "given_name")
}

// --- SetPersonMiddleName / UnsetPersonMiddleName ---

type SetPersonMiddleName struct {
	PersonID ID `json:"person_id"`
	Val      string
	userID   *ID
	role     *string
}

func (m *SetPersonMiddleName) name() string       { return "set:person_middle_name" }
func (m *SetPersonMiddleName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonMiddleName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.MiddleName == m.Val {
		return nil, nil
	}
	return applySetPersonField(state, m.PersonID, "middle_name", &m.userID, &m.role, userID, role)
}
func (m *SetPersonMiddleName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetPersonField(ctx, tx, revID, m.PersonID, "middle_name", m.Val, m.userID, m.role)
}

type UnsetPersonMiddleName struct{ PersonID ID }

func (m *UnsetPersonMiddleName) name() string       { return "unset:person_middle_name" }
func (m *UnsetPersonMiddleName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *UnsetPersonMiddleName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.MiddleName == "" {
		return nil, nil
	}
	return applyUnsetPersonField(state, role, m.PersonID, "middle_name")
}
func (m *UnsetPersonMiddleName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetPersonField(ctx, tx, revID, m.PersonID, "middle_name")
}

// --- SetPersonFamilyName / UnsetPersonFamilyName ---

type SetPersonFamilyName struct {
	PersonID ID `json:"person_id"`
	Val      string
	userID   *ID
	role     *string
}

func (m *SetPersonFamilyName) name() string       { return "set:person_family_name" }
func (m *SetPersonFamilyName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonFamilyName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.FamilyName == m.Val {
		return nil, nil
	}
	return applySetPersonField(state, m.PersonID, "family_name", &m.userID, &m.role, userID, role)
}
func (m *SetPersonFamilyName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetPersonField(ctx, tx, revID, m.PersonID, "family_name", m.Val, m.userID, m.role)
}

type UnsetPersonFamilyName struct{ PersonID ID }

func (m *UnsetPersonFamilyName) name() string       { return "unset:person_family_name" }
func (m *UnsetPersonFamilyName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *UnsetPersonFamilyName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && p.FamilyName == "" {
		return nil, nil
	}
	return applyUnsetPersonField(state, role, m.PersonID, "family_name")
}
func (m *UnsetPersonFamilyName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetPersonField(ctx, tx, revID, m.PersonID, "family_name")
}

// ============================================================
// Set / Unset updaters for person collectives
// ============================================================

// --- SetPersonIdentifiers / UnsetPersonIdentifiers ---

type SetPersonIdentifiers struct {
	PersonID    ID
	Identifiers []Identifier `json:"identifiers"`
	userID      *ID
	role        *string
}

func (m *SetPersonIdentifiers) name() string       { return "set:person_identifiers" }
func (m *SetPersonIdentifiers) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && slicesEqual(p.Identifiers, m.Identifiers) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "identifiers", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "identifiers", revID); err != nil {
		return fmt.Errorf("SetPersonIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.PersonID, "identifiers"); err != nil {
		return fmt.Errorf("SetPersonIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if _, err := writePersonAssertion(ctx, tx, revID, m.PersonID, "identifiers", ident, false, nil, m.userID, m.role); err != nil {
			return fmt.Errorf("SetPersonIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetPersonIdentifiers struct{ PersonID ID }

func (m *UnsetPersonIdentifiers) name() string       { return "unset:person_identifiers" }
func (m *UnsetPersonIdentifiers) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *UnsetPersonIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && len(p.Identifiers) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "identifiers", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *UnsetPersonIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "identifiers", revID); err != nil {
		return fmt.Errorf("UnsetPersonIdentifiers: %w", err)
	}
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
	role          *string
}

func (m *SetPersonOrganizations) name() string       { return "set:person_organizations" }
func (m *SetPersonOrganizations) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *SetPersonOrganizations) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && personOrganizationsEqual(p.Organizations, m.Organizations) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "organizations") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "organizations", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *SetPersonOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "organizations", revID); err != nil {
		return fmt.Errorf("SetPersonOrganizations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.PersonID, "organizations"); err != nil {
		return fmt.Errorf("SetPersonOrganizations: %w", err)
	}
	for _, org := range m.Organizations {
		assertionID, err := writePersonAssertion(ctx, tx, revID, m.PersonID, "organizations", nil, false, nil, m.userID, m.role)
		if err != nil {
			return fmt.Errorf("SetPersonOrganizations: %w", err)
		}
		if err := writePersonOrganization(ctx, tx, assertionID, org.OrganizationID, nil, nil); err != nil {
			return fmt.Errorf("SetPersonOrganizations: %w", err)
		}
	}
	return nil
}

type UnsetPersonOrganizations struct{ PersonID ID }

func (m *UnsetPersonOrganizations) name() string { return "unset:person_organizations" }
func (m *UnsetPersonOrganizations) needs() updateNeeds {
	return updateNeeds{personIDs: []ID{m.PersonID}}
}
func (m *UnsetPersonOrganizations) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if p := state.people[m.PersonID]; p != nil && len(p.Organizations) == 0 {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "organizations") {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "organizations", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *UnsetPersonOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "organizations", revID); err != nil {
		return fmt.Errorf("UnsetPersonOrganizations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.PersonID); err != nil {
		return fmt.Errorf("UnsetPersonOrganizations: %w", err)
	}
	return nil
}

// ============================================================
// Hide updaters for person fields
// ============================================================

// --- HidePersonGivenName ---

type HidePersonGivenName struct {
	PersonID ID
	userID   *ID
	role     *string
}

func (m *HidePersonGivenName) name() string       { return "hide:person_given_name" }
func (m *HidePersonGivenName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *HidePersonGivenName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHidePersonField(state, m.PersonID, "given_name", &m.userID, &m.role, userID, role)
}
func (m *HidePersonGivenName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHidePersonField(ctx, tx, revID, m.PersonID, "given_name", m.userID, m.role)
}

// --- HidePersonMiddleName ---

type HidePersonMiddleName struct {
	PersonID ID
	userID   *ID
	role     *string
}

func (m *HidePersonMiddleName) name() string       { return "hide:person_middle_name" }
func (m *HidePersonMiddleName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *HidePersonMiddleName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHidePersonField(state, m.PersonID, "middle_name", &m.userID, &m.role, userID, role)
}
func (m *HidePersonMiddleName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHidePersonField(ctx, tx, revID, m.PersonID, "middle_name", m.userID, m.role)
}

// --- HidePersonFamilyName ---

type HidePersonFamilyName struct {
	PersonID ID
	userID   *ID
	role     *string
}

func (m *HidePersonFamilyName) name() string       { return "hide:person_family_name" }
func (m *HidePersonFamilyName) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *HidePersonFamilyName) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHidePersonField(state, m.PersonID, "family_name", &m.userID, &m.role, userID, role)
}
func (m *HidePersonFamilyName) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHidePersonField(ctx, tx, revID, m.PersonID, "family_name", m.userID, m.role)
}

// --- HidePersonIdentifiers ---

type HidePersonIdentifiers struct {
	PersonID ID
	userID   *ID
	role     *string
}

func (m *HidePersonIdentifiers) name() string       { return "hide:person_identifiers" }
func (m *HidePersonIdentifiers) needs() updateNeeds { return updateNeeds{personIDs: []ID{m.PersonID}} }
func (m *HidePersonIdentifiers) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.personAssertions[m.PersonID], "identifiers") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "identifiers") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "identifiers", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *HidePersonIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "identifiers", revID); err != nil {
		return fmt.Errorf("HidePersonIdentifiers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.PersonID, "identifiers"); err != nil {
		return fmt.Errorf("HidePersonIdentifiers: %w", err)
	}
	_, err := writePersonAssertion(ctx, tx, revID, m.PersonID, "identifiers", nil, true, nil, m.userID, m.role)
	return err
}

// --- HidePersonOrganizations ---

type HidePersonOrganizations struct {
	PersonID ID
	userID   *ID
	role     *string
}

func (m *HidePersonOrganizations) name() string { return "hide:person_organizations" }
func (m *HidePersonOrganizations) needs() updateNeeds {
	return updateNeeds{personIDs: []ID{m.PersonID}}
}
func (m *HidePersonOrganizations) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.personAssertions[m.PersonID], "organizations") {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.personAssertions[m.PersonID], "organizations") {
			return nil, ErrCuratorLock
		}
	}
	m.userID = userID
	m.role = &role
	return &updateEffect{
		recordType: RecordTypePerson,
		recordID:   m.PersonID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_person_assertions", "person_id", m.PersonID, "organizations", "person_source_id", "bbl_person_sources", priorities)
		},
	}, nil
}
func (m *HidePersonOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if err := logPersonHistory(ctx, tx, m.PersonID, "organizations", revID); err != nil {
		return fmt.Errorf("HidePersonOrganizations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_person_assertions WHERE person_id = $1 AND field = $2 AND user_id IS NOT NULL`, m.PersonID, "organizations"); err != nil {
		return fmt.Errorf("HidePersonOrganizations: %w", err)
	}
	_, err := writePersonAssertion(ctx, tx, revID, m.PersonID, "organizations", nil, true, nil, m.userID, m.role)
	return err
}
