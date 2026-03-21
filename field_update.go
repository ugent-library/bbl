package bbl

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Set asserts a value for a field. Generic — works for any entity and field.
type Set struct {
	RecordType string `json:"record_type"`
	RecordID   ID     `json:"id"`
	Field      string `json:"field"`
	Val        any    `json:"val"`
}

func (m *Set) name() string { return "set:" + m.RecordType + "." + m.Field }

func (m *Set) needs() updateNeeds {
	return needsForEntity(m.RecordType, m.RecordID)
}

func (m *Set) apply(state updateState, user *User) (*updateEffect, error) {
	ft, err := resolveFieldType(m.RecordType, m.Field)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", m.name(), err)
	}

	rs := state.records[m.RecordID]

	// Noop: compare against current pinned value.
	if rs != nil {
		if cached, ok := rs.fields[m.Field]; ok && ft.equal(cached, m.Val) {
			return nil, nil
		}
	}

	// Curator lock.
	if rs != nil {
		if fs := rs.assertions[m.Field]; fs != nil {
			if user.Role != "curator" && fs.userID != nil && fs.role == "curator" {
				return nil, ErrCuratorLock
			}
		}
	}

	// Mutate record state.
	if rs != nil {
		rs.fields[m.Field] = m.Val
	}

	return &updateEffect{
		recordType: m.RecordType,
		recordID:   m.RecordID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx,
				assertionsTable(m.RecordType), entityIDCol(m.RecordType),
				m.RecordID, m.Field,
				sourceIDCol(m.RecordType), sourceTable(m.RecordType),
				priorities)
		},
	}, nil
}

func (m *Set) write(revID int64, user *User) (string, []any) {
	return "", nil // field ops use executeFieldWrites
}

// Hide asserts that a field intentionally has no value.
type Hide struct {
	RecordType string `json:"record_type"`
	RecordID   ID     `json:"id"`
	Field      string `json:"field"`
}

func (m *Hide) name() string { return "hide:" + m.RecordType + "." + m.Field }

func (m *Hide) needs() updateNeeds {
	return needsForEntity(m.RecordType, m.RecordID)
}

func (m *Hide) apply(state updateState, user *User) (*updateEffect, error) {
	if _, err := resolveFieldType(m.RecordType, m.Field); err != nil {
		return nil, fmt.Errorf("%s: %w", m.name(), err)
	}

	rs := state.records[m.RecordID]
	if rs != nil {
		if fs := rs.assertions[m.Field]; fs != nil {
			// Noop: already hidden.
			if fs.hidden {
				return nil, nil
			}
			// Curator lock.
			if user.Role != "curator" && fs.userID != nil && fs.role == "curator" {
				return nil, ErrCuratorLock
			}
		}
		delete(rs.fields, m.Field)
	}

	return &updateEffect{
		recordType: m.RecordType,
		recordID:   m.RecordID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx,
				assertionsTable(m.RecordType), entityIDCol(m.RecordType),
				m.RecordID, m.Field,
				sourceIDCol(m.RecordType), sourceTable(m.RecordType),
				priorities)
		},
	}, nil
}

func (m *Hide) write(revID int64, user *User) (string, []any) {
	return "", nil // field ops use executeFieldWrites
}

// Unset withdraws the human assertion for a field.
type Unset struct {
	RecordType string `json:"record_type"`
	RecordID   ID     `json:"id"`
	Field      string `json:"field"`
}

func (m *Unset) name() string { return "unset:" + m.RecordType + "." + m.Field }

func (m *Unset) needs() updateNeeds {
	return needsForEntity(m.RecordType, m.RecordID)
}

func (m *Unset) apply(state updateState, user *User) (*updateEffect, error) {
	if _, err := resolveFieldType(m.RecordType, m.Field); err != nil {
		return nil, fmt.Errorf("%s: %w", m.name(), err)
	}

	rs := state.records[m.RecordID]

	// Noop: no human assertion exists.
	if rs == nil {
		return nil, nil
	}
	fs := rs.assertions[m.Field]
	if fs == nil || fs.userID == nil {
		return nil, nil
	}

	// Curator lock.
	if user.Role != "curator" && fs.role == "curator" {
		return nil, ErrCuratorLock
	}

	delete(rs.fields, m.Field)

	return &updateEffect{
		recordType: m.RecordType,
		recordID:   m.RecordID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx,
				assertionsTable(m.RecordType), entityIDCol(m.RecordType),
				m.RecordID, m.Field,
				sourceIDCol(m.RecordType), sourceTable(m.RecordType),
				priorities)
		},
	}, nil
}

func (m *Unset) write(revID int64, user *User) (string, []any) {
	return "", nil // field ops use executeFieldWrites
}

// --- helpers ---

func needsForEntity(recordType string, id ID) updateNeeds {
	switch recordType {
	case "work":
		return updateNeeds{workIDs: []ID{id}}
	case "person":
		return updateNeeds{personIDs: []ID{id}}
	case "project":
		return updateNeeds{projectIDs: []ID{id}}
	case "organization":
		return updateNeeds{organizationIDs: []ID{id}}
	default:
		return updateNeeds{}
	}
}

func assertionsTable(rt string) string {
	switch rt {
	case "work":
		return "bbl_work_assertions"
	case "person":
		return "bbl_person_assertions"
	case "project":
		return "bbl_project_assertions"
	case "organization":
		return "bbl_organization_assertions"
	}
	return ""
}

func entityIDCol(rt string) string {
	switch rt {
	case "work":
		return "work_id"
	case "person":
		return "person_id"
	case "project":
		return "project_id"
	case "organization":
		return "organization_id"
	}
	return ""
}

func sourceIDCol(rt string) string {
	switch rt {
	case "work":
		return "work_source_id"
	case "person":
		return "person_source_id"
	case "project":
		return "project_source_id"
	case "organization":
		return "organization_source_id"
	}
	return ""
}

func sourceTable(rt string) string {
	switch rt {
	case "work":
		return "bbl_work_sources"
	case "person":
		return "bbl_person_sources"
	case "project":
		return "bbl_project_sources"
	case "organization":
		return "bbl_organization_sources"
	}
	return ""
}

type fieldState struct {
	val    any
	hidden bool
	userID *ID
	role   string
}

