package bbl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for relation tables ---
// These insert value rows linked to an assertion via assertion_id.
// The assertion row must be created first.

// writeWorkAssertion creates an assertion row. Used by both Set mutations and import.
func writeWorkAssertion(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, hidden bool, workSourceID *ID, userID *ID, role *string) (int64, error) {
	var valJSON []byte
	if val != nil {
		var err error
		valJSON, err = json.Marshal(val)
		if err != nil {
			return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
		}
	}
	var id int64
	err := tx.QueryRow(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, hidden, work_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		revID, workID, field, valJSON, hidden, workSourceID, userID, role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("writeWorkAssertion(%s): %w", field, err)
	}
	return id, nil
}

func writeWorkIdentifier(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_identifiers (id, assertion_id, work_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, scheme, val)
	if err != nil {
		return fmt.Errorf("writeWorkIdentifier: %w", err)
	}
	return nil
}

func writeWorkClassification(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, scheme, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_classifications (id, assertion_id, work_id, scheme, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, scheme, val)
	if err != nil {
		return fmt.Errorf("writeWorkClassification: %w", err)
	}
	return nil
}

func writeWorkContributor(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, position int, kind string, personID *ID, name, givenName, familyName string, roles []string) error {
	if kind == "" {
		kind = "person"
	}
	if name == "" {
		name = strings.TrimSpace(givenName + " " + familyName)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_contributors
		    (id, assertion_id, work_id, position, kind, person_id, name, given_name, family_name, roles)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, assertionID, workID, position, kind, personID,
		name, nilIfEmpty(givenName), nilIfEmpty(familyName),
		roles)
	if err != nil {
		return fmt.Errorf("writeWorkContributor: %w", err)
	}
	return nil
}

func writeWorkTitle(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_titles (id, assertion_id, work_id, lang, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, lang, val)
	if err != nil {
		return fmt.Errorf("writeWorkTitle: %w", err)
	}
	return nil
}

func writeWorkAbstract(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_abstracts (id, assertion_id, work_id, lang, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, lang, val)
	if err != nil {
		return fmt.Errorf("writeWorkAbstract: %w", err)
	}
	return nil
}

func writeWorkLaySummary(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, lang, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_lay_summaries (id, assertion_id, work_id, lang, val)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, lang, val)
	if err != nil {
		return fmt.Errorf("writeWorkLaySummary: %w", err)
	}
	return nil
}

func writeWorkNote(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, val, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_notes (id, assertion_id, work_id, val, kind)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, val, nilIfEmpty(kind))
	if err != nil {
		return fmt.Errorf("writeWorkNote: %w", err)
	}
	return nil
}

func writeWorkKeyword(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID ID, val string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_keywords (id, assertion_id, work_id, val)
		VALUES ($1, $2, $3, $4)`,
		id, assertionID, workID, val)
	if err != nil {
		return fmt.Errorf("writeWorkKeyword: %w", err)
	}
	return nil
}

func writeWorkProject(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID, projectID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_projects (id, assertion_id, work_id, project_id)
		VALUES ($1, $2, $3, $4)`,
		id, assertionID, workID, projectID)
	if err != nil {
		return fmt.Errorf("writeWorkProject: %w", err)
	}
	return nil
}

func writeWorkOrganization(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID, orgID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_organizations (id, assertion_id, work_id, organization_id)
		VALUES ($1, $2, $3, $4)`,
		id, assertionID, workID, orgID)
	if err != nil {
		return fmt.Errorf("writeWorkOrganization: %w", err)
	}
	return nil
}

func writeWorkRel(ctx context.Context, tx pgx.Tx, id ID, assertionID int64, workID, relatedWorkID ID, kind string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_rels (id, assertion_id, work_id, related_work_id, kind)
		VALUES ($1, $2, $3, $4, $5)`,
		id, assertionID, workID, relatedWorkID, kind)
	if err != nil {
		return fmt.Errorf("writeWorkRel: %w", err)
	}
	return nil
}

// ============================================================
// Set / Unset mutations for work collectives
// ============================================================

// --- SetWorkTitles (no delete — required) ---

type SetWorkTitles struct {
	WorkID ID     `json:"work_id"`
	Titles []Title `json:"titles"`
	userID *ID
}

func (m *SetWorkTitles) mutationName() string { return "set_work_titles" }
func (m *SetWorkTitles) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkTitles) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "titles", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkTitles) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "titles", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkTitles: %w", err)
	}
	for _, t := range m.Titles {
		if err := writeWorkTitle(ctx, tx, newID(), assertionID, m.WorkID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetWorkTitles: %w", err)
		}
	}
	return nil
}

// --- SetWorkAbstracts / UnsetWorkAbstracts ---

type SetWorkAbstracts struct {
	WorkID    ID
	Abstracts []Text `json:"abstracts"`
	userID    *ID
}

func (m *SetWorkAbstracts) mutationName() string { return "set_work_abstracts" }
func (m *SetWorkAbstracts) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkAbstracts) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "abstracts", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkAbstracts) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "abstracts", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkAbstracts: %w", err)
	}
	for _, t := range m.Abstracts {
		if err := writeWorkAbstract(ctx, tx, newID(), assertionID, m.WorkID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetWorkAbstracts: %w", err)
		}
	}
	return nil
}

type UnsetWorkAbstracts struct{ WorkID ID }

func (m *UnsetWorkAbstracts) mutationName() string { return "unset_work_abstracts" }
func (m *UnsetWorkAbstracts) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkAbstracts) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "abstracts", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkAbstracts) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'abstracts' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkAbstracts: %w", err)
	}
	return nil
}

// --- SetWorkLaySummaries / UnsetWorkLaySummaries ---

type SetWorkLaySummaries struct {
	WorkID       ID
	LaySummaries []Text `json:"lay_summaries"`
	userID       *ID
}

func (m *SetWorkLaySummaries) mutationName() string { return "set_work_lay_summaries" }
func (m *SetWorkLaySummaries) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkLaySummaries) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "lay_summaries", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkLaySummaries) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "lay_summaries", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkLaySummaries: %w", err)
	}
	for _, t := range m.LaySummaries {
		if err := writeWorkLaySummary(ctx, tx, newID(), assertionID, m.WorkID, t.Lang, t.Val); err != nil {
			return fmt.Errorf("SetWorkLaySummaries: %w", err)
		}
	}
	return nil
}

type UnsetWorkLaySummaries struct{ WorkID ID }

func (m *UnsetWorkLaySummaries) mutationName() string { return "unset_work_lay_summaries" }
func (m *UnsetWorkLaySummaries) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkLaySummaries) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "lay_summaries", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkLaySummaries) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'lay_summaries' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkLaySummaries: %w", err)
	}
	return nil
}

// --- SetWorkNotes / UnsetWorkNotes ---

type SetWorkNotes struct {
	WorkID ID     `json:"work_id"`
	Notes  []Note `json:"notes"`
	userID *ID
}

func (m *SetWorkNotes) mutationName() string { return "set_work_notes" }
func (m *SetWorkNotes) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkNotes) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "notes", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkNotes) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "notes", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkNotes: %w", err)
	}
	for _, n := range m.Notes {
		if err := writeWorkNote(ctx, tx, newID(), assertionID, m.WorkID, n.Val, n.Kind); err != nil {
			return fmt.Errorf("SetWorkNotes: %w", err)
		}
	}
	return nil
}

type UnsetWorkNotes struct{ WorkID ID }

func (m *UnsetWorkNotes) mutationName() string { return "unset_work_notes" }
func (m *UnsetWorkNotes) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkNotes) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "notes", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkNotes) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'notes' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkNotes: %w", err)
	}
	return nil
}

// --- SetWorkKeywords / UnsetWorkKeywords ---

type SetWorkKeywords struct {
	WorkID   ID
	Keywords []Keyword `json:"keywords"`
	userID   *ID
}

func (m *SetWorkKeywords) mutationName() string { return "set_work_keywords" }
func (m *SetWorkKeywords) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkKeywords) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "keywords", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkKeywords) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "keywords", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkKeywords: %w", err)
	}
	for _, k := range m.Keywords {
		if err := writeWorkKeyword(ctx, tx, newID(), assertionID, m.WorkID, k.Val); err != nil {
			return fmt.Errorf("SetWorkKeywords: %w", err)
		}
	}
	return nil
}

type UnsetWorkKeywords struct{ WorkID ID }

func (m *UnsetWorkKeywords) mutationName() string { return "unset_work_keywords" }
func (m *UnsetWorkKeywords) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkKeywords) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "keywords", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkKeywords) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'keywords' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkKeywords: %w", err)
	}
	return nil
}

// --- SetWorkIdentifiers / UnsetWorkIdentifiers ---

type SetWorkIdentifiers struct {
	WorkID      ID
	Identifiers []WorkIdentifier
	userID      *ID
}

func (m *SetWorkIdentifiers) mutationName() string { return "set_work_identifiers" }
func (m *SetWorkIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "identifiers", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "identifiers", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkIdentifiers: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeWorkIdentifier(ctx, tx, newID(), assertionID, m.WorkID, ident.Scheme, ident.Val); err != nil {
			return fmt.Errorf("SetWorkIdentifiers: %w", err)
		}
	}
	return nil
}

type UnsetWorkIdentifiers struct{ WorkID ID }

func (m *UnsetWorkIdentifiers) mutationName() string { return "unset_work_identifiers" }
func (m *UnsetWorkIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkIdentifiers) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "identifiers", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkIdentifiers) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'identifiers' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkIdentifiers: %w", err)
	}
	return nil
}

// --- SetWorkClassifications / UnsetWorkClassifications ---

type SetWorkClassifications struct {
	WorkID          ID
	Classifications []WorkClassification
	userID          *ID
}

func (m *SetWorkClassifications) mutationName() string { return "set_work_classifications" }
func (m *SetWorkClassifications) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkClassifications) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "classifications", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkClassifications) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "classifications", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkClassifications: %w", err)
	}
	for _, c := range m.Classifications {
		if err := writeWorkClassification(ctx, tx, newID(), assertionID, m.WorkID, c.Scheme, c.Val); err != nil {
			return fmt.Errorf("SetWorkClassifications: %w", err)
		}
	}
	return nil
}

type UnsetWorkClassifications struct{ WorkID ID }

func (m *UnsetWorkClassifications) mutationName() string { return "unset_work_classifications" }
func (m *UnsetWorkClassifications) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkClassifications) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "classifications", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkClassifications) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'classifications' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkClassifications: %w", err)
	}
	return nil
}

// --- SetWorkContributors / UnsetWorkContributors ---

type SetWorkContributors struct {
	WorkID       ID
	Contributors []WorkContributor `json:"contributors"`
	userID       *ID
}

func (m *SetWorkContributors) mutationName() string { return "set_work_contributors" }
func (m *SetWorkContributors) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkContributors) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "contributors", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkContributors) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "contributors", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkContributors: %w", err)
	}
	for i, c := range m.Contributors {
		if err := writeWorkContributor(ctx, tx, newID(), assertionID, m.WorkID, i, c.Kind, c.PersonID, c.Name, c.GivenName, c.FamilyName, c.Roles); err != nil {
			return fmt.Errorf("SetWorkContributors: %w", err)
		}
	}
	return nil
}

type UnsetWorkContributors struct{ WorkID ID }

func (m *UnsetWorkContributors) mutationName() string { return "unset_work_contributors" }
func (m *UnsetWorkContributors) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkContributors) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "contributors", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkContributors) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'contributors' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkContributors: %w", err)
	}
	return nil
}

// --- SetWorkProjects / UnsetWorkProjects ---

type SetWorkProjects struct {
	WorkID   ID
	Projects []ID `json:"projects"`
	userID   *ID
}

func (m *SetWorkProjects) mutationName() string { return "set_work_projects" }
func (m *SetWorkProjects) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkProjects) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "projects", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkProjects) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "projects", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkProjects: %w", err)
	}
	for _, pid := range m.Projects {
		if err := writeWorkProject(ctx, tx, newID(), assertionID, m.WorkID, pid); err != nil {
			return fmt.Errorf("SetWorkProjects: %w", err)
		}
	}
	return nil
}

type UnsetWorkProjects struct{ WorkID ID }

func (m *UnsetWorkProjects) mutationName() string { return "unset_work_projects" }
func (m *UnsetWorkProjects) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkProjects) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "projects", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkProjects) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'projects' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkProjects: %w", err)
	}
	return nil
}

// --- SetWorkOrganizations / UnsetWorkOrganizations ---

type SetWorkOrganizations struct {
	WorkID        ID
	Organizations []ID `json:"organizations"`
	userID        *ID
}

func (m *SetWorkOrganizations) mutationName() string { return "set_work_organizations" }
func (m *SetWorkOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkOrganizations) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "organizations", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "organizations", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkOrganizations: %w", err)
	}
	for _, oid := range m.Organizations {
		if err := writeWorkOrganization(ctx, tx, newID(), assertionID, m.WorkID, oid); err != nil {
			return fmt.Errorf("SetWorkOrganizations: %w", err)
		}
	}
	return nil
}

type UnsetWorkOrganizations struct{ WorkID ID }

func (m *UnsetWorkOrganizations) mutationName() string { return "unset_work_organizations" }
func (m *UnsetWorkOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkOrganizations) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "organizations", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkOrganizations) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'organizations' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkOrganizations: %w", err)
	}
	return nil
}

// --- SetWorkRels / UnsetWorkRels ---

type SetWorkRels struct {
	WorkID ID     `json:"work_id"`
	Rels []struct {
		RelatedWorkID ID     `json:"related_work_id"`
		Kind          string `json:"kind"`
	} `json:"rels"`
	userID *ID
}

func (m *SetWorkRels) mutationName() string { return "set_work_rels" }
func (m *SetWorkRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkRels) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	m.userID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "rels", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	assertionID, err := writeWorkAssertion(ctx, tx, revID, m.WorkID, "rels", nil, false, nil, m.userID, nil)
	if err != nil {
		return fmt.Errorf("SetWorkRels: %w", err)
	}
	for _, r := range m.Rels {
		if err := writeWorkRel(ctx, tx, newID(), assertionID, m.WorkID, r.RelatedWorkID, r.Kind); err != nil {
			return fmt.Errorf("SetWorkRels: %w", err)
		}
	}
	return nil
}

type UnsetWorkRels struct{ WorkID ID }

func (m *UnsetWorkRels) mutationName() string { return "unset_work_rels" }
func (m *UnsetWorkRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkRels) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", m.WorkID, "rels", "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *UnsetWorkRels) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = 'rels' AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("UnsetWorkRels: %w", err)
	}
	return nil
}
