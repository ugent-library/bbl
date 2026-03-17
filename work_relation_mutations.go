package bbl

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers for relation tables ---
// These are used by both Set mutations (human path) and import.

func writeWorkIdentifier(ctx context.Context, tx pgx.Tx, id, workID ID, scheme, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_identifiers (id, work_id, scheme, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, scheme, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkIdentifier: %w", err)
	}
	return nil
}

func writeWorkClassification(ctx context.Context, tx pgx.Tx, id, workID ID, scheme, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_classifications (id, work_id, scheme, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, scheme, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkClassification: %w", err)
	}
	return nil
}

func writeWorkContributor(ctx context.Context, tx pgx.Tx, id, workID ID, position int, kind string, personID *ID, name, givenName, familyName string, roles []string, workSourceID *ID, userID *ID) error {
	if kind == "" {
		kind = "person"
	}
	if name == "" {
		name = strings.TrimSpace(givenName + " " + familyName)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_contributors
		    (id, work_id, position, kind, person_id, name, given_name, family_name, roles, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		id, workID, position, kind, personID,
		name, nilIfEmpty(givenName), nilIfEmpty(familyName),
		roles, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkContributor: %w", err)
	}
	return nil
}

func writeWorkTitle(ctx context.Context, tx pgx.Tx, id, workID ID, lang, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_titles (id, work_id, lang, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, lang, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkTitle: %w", err)
	}
	return nil
}

func writeWorkAbstract(ctx context.Context, tx pgx.Tx, id, workID ID, lang, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_abstracts (id, work_id, lang, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, lang, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkAbstract: %w", err)
	}
	return nil
}

func writeWorkLaySummary(ctx context.Context, tx pgx.Tx, id, workID ID, lang, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_lay_summaries (id, work_id, lang, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, lang, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkLaySummary: %w", err)
	}
	return nil
}

func writeWorkNote(ctx context.Context, tx pgx.Tx, id, workID ID, val, kind string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_notes (id, work_id, val, kind, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, val, nilIfEmpty(kind), workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkNote: %w", err)
	}
	return nil
}

func writeWorkKeyword(ctx context.Context, tx pgx.Tx, id, workID ID, val string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_keywords (id, work_id, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5)`,
		id, workID, val, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkKeyword: %w", err)
	}
	return nil
}

func writeWorkProject(ctx context.Context, tx pgx.Tx, id, workID, projectID ID, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_projects (id, work_id, project_id, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5)`,
		id, workID, projectID, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkProject: %w", err)
	}
	return nil
}

func writeWorkOrganization(ctx context.Context, tx pgx.Tx, id, workID, orgID ID, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_organizations (id, work_id, organization_id, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5)`,
		id, workID, orgID, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkOrganization: %w", err)
	}
	return nil
}

func writeWorkRel(ctx context.Context, tx pgx.Tx, id, workID, relatedWorkID ID, kind string, workSourceID *ID, userID *ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_rels (id, work_id, related_work_id, kind, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, relatedWorkID, kind, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeWorkRel: %w", err)
	}
	return nil
}

// ============================================================
// Set / Delete mutations for work collectives
// ============================================================

// --- SetWorkTitles (no delete — required) ---

type SetWorkTitles struct {
	WorkID ID
	Titles []Title
	userID *ID
}

func (m *SetWorkTitles) mutationName() string { return "SetWorkTitles" }
func (m *SetWorkTitles) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkTitles) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Titles []Title }{m.Titles}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_titles", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkTitles) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_titles WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkTitles: delete: %w", err)
	}
	for _, t := range m.Titles {
		if err := writeWorkTitle(ctx, tx, newID(), m.WorkID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkTitles: %w", err)
		}
	}
	return nil
}

// --- SetWorkAbstracts / DeleteWorkAbstracts ---

type SetWorkAbstracts struct {
	WorkID    ID
	Abstracts []Text
	userID    *ID
}

func (m *SetWorkAbstracts) mutationName() string { return "SetWorkAbstracts" }
func (m *SetWorkAbstracts) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkAbstracts) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Abstracts []Text }{m.Abstracts}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_abstracts", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkAbstracts) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_abstracts WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkAbstracts: delete: %w", err)
	}
	for _, t := range m.Abstracts {
		if err := writeWorkAbstract(ctx, tx, newID(), m.WorkID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkAbstracts: %w", err)
		}
	}
	return nil
}

type DeleteWorkAbstracts struct{ WorkID ID }

func (m *DeleteWorkAbstracts) mutationName() string { return "DeleteWorkAbstracts" }
func (m *DeleteWorkAbstracts) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkAbstracts) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_abstracts", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkAbstracts) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_abstracts WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkAbstracts: %w", err)
	}
	return nil
}

// --- SetWorkLaySummaries / DeleteWorkLaySummaries ---

type SetWorkLaySummaries struct {
	WorkID       ID
	LaySummaries []Text
	userID       *ID
}

func (m *SetWorkLaySummaries) mutationName() string { return "SetWorkLaySummaries" }
func (m *SetWorkLaySummaries) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkLaySummaries) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ LaySummaries []Text }{m.LaySummaries}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_lay_summaries", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkLaySummaries) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_lay_summaries WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkLaySummaries: delete: %w", err)
	}
	for _, t := range m.LaySummaries {
		if err := writeWorkLaySummary(ctx, tx, newID(), m.WorkID, t.Lang, t.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkLaySummaries: %w", err)
		}
	}
	return nil
}

type DeleteWorkLaySummaries struct{ WorkID ID }

func (m *DeleteWorkLaySummaries) mutationName() string { return "DeleteWorkLaySummaries" }
func (m *DeleteWorkLaySummaries) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkLaySummaries) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_lay_summaries", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkLaySummaries) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_lay_summaries WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkLaySummaries: %w", err)
	}
	return nil
}

// --- SetWorkNotes / DeleteWorkNotes ---

type SetWorkNotes struct {
	WorkID ID
	Notes  []Note
	userID *ID
}

func (m *SetWorkNotes) mutationName() string { return "SetWorkNotes" }
func (m *SetWorkNotes) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkNotes) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Notes []Note }{m.Notes}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_notes", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkNotes) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_notes WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkNotes: delete: %w", err)
	}
	for _, n := range m.Notes {
		if err := writeWorkNote(ctx, tx, newID(), m.WorkID, n.Val, n.Kind, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkNotes: %w", err)
		}
	}
	return nil
}

type DeleteWorkNotes struct{ WorkID ID }

func (m *DeleteWorkNotes) mutationName() string { return "DeleteWorkNotes" }
func (m *DeleteWorkNotes) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkNotes) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_notes", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkNotes) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_notes WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkNotes: %w", err)
	}
	return nil
}

// --- SetWorkKeywords / DeleteWorkKeywords ---

type SetWorkKeywords struct {
	WorkID   ID
	Keywords []Keyword
	userID   *ID
}

func (m *SetWorkKeywords) mutationName() string { return "SetWorkKeywords" }
func (m *SetWorkKeywords) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkKeywords) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Keywords []Keyword }{m.Keywords}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_keywords", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkKeywords) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_keywords WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkKeywords: delete: %w", err)
	}
	for _, k := range m.Keywords {
		if err := writeWorkKeyword(ctx, tx, newID(), m.WorkID, k.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkKeywords: %w", err)
		}
	}
	return nil
}

type DeleteWorkKeywords struct{ WorkID ID }

func (m *DeleteWorkKeywords) mutationName() string { return "DeleteWorkKeywords" }
func (m *DeleteWorkKeywords) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkKeywords) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_keywords", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkKeywords) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_keywords WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkKeywords: %w", err)
	}
	return nil
}

// --- SetWorkIdentifiers / DeleteWorkIdentifiers ---

type SetWorkIdentifiers struct {
	WorkID      ID
	Identifiers []WorkIdentifier
	userID      *ID
}

func (m *SetWorkIdentifiers) mutationName() string { return "SetWorkIdentifiers" }
func (m *SetWorkIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Identifiers []WorkIdentifier }{m.Identifiers}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_identifiers", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_identifiers WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkIdentifiers: delete: %w", err)
	}
	for _, ident := range m.Identifiers {
		if err := writeWorkIdentifier(ctx, tx, newID(), m.WorkID, ident.Scheme, ident.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkIdentifiers: %w", err)
		}
	}
	return nil
}

type DeleteWorkIdentifiers struct{ WorkID ID }

func (m *DeleteWorkIdentifiers) mutationName() string { return "DeleteWorkIdentifiers" }
func (m *DeleteWorkIdentifiers) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkIdentifiers) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_identifiers", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkIdentifiers) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_identifiers WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkIdentifiers: %w", err)
	}
	return nil
}

// --- SetWorkClassifications / DeleteWorkClassifications ---

type SetWorkClassifications struct {
	WorkID          ID
	Classifications []WorkClassification
	userID          *ID
}

func (m *SetWorkClassifications) mutationName() string { return "SetWorkClassifications" }
func (m *SetWorkClassifications) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkClassifications) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Classifications []WorkClassification }{m.Classifications}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_classifications", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkClassifications) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_classifications WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkClassifications: delete: %w", err)
	}
	for _, c := range m.Classifications {
		if err := writeWorkClassification(ctx, tx, newID(), m.WorkID, c.Scheme, c.Val, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkClassifications: %w", err)
		}
	}
	return nil
}

type DeleteWorkClassifications struct{ WorkID ID }

func (m *DeleteWorkClassifications) mutationName() string { return "DeleteWorkClassifications" }
func (m *DeleteWorkClassifications) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkClassifications) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_classifications", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkClassifications) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_classifications WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkClassifications: %w", err)
	}
	return nil
}

// --- SetWorkContributors / DeleteWorkContributors ---

type SetWorkContributors struct {
	WorkID       ID
	Contributors []WorkContributor
	userID       *ID
}

func (m *SetWorkContributors) mutationName() string { return "SetWorkContributors" }
func (m *SetWorkContributors) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkContributors) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Contributors []WorkContributor }{m.Contributors}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_contributors", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkContributors) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_contributors WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkContributors: delete: %w", err)
	}
	for i, c := range m.Contributors {
		if err := writeWorkContributor(ctx, tx, newID(), m.WorkID, i, c.Kind, c.PersonID, c.Name, c.GivenName, c.FamilyName, c.Roles, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkContributors: %w", err)
		}
	}
	return nil
}

type DeleteWorkContributors struct{ WorkID ID }

func (m *DeleteWorkContributors) mutationName() string { return "DeleteWorkContributors" }
func (m *DeleteWorkContributors) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkContributors) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_contributors", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkContributors) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_contributors WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkContributors: %w", err)
	}
	return nil
}

// --- SetWorkProjects / DeleteWorkProjects ---

type SetWorkProjects struct {
	WorkID   ID
	Projects []ID
	userID   *ID
}

func (m *SetWorkProjects) mutationName() string { return "SetWorkProjects" }
func (m *SetWorkProjects) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkProjects) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Projects []ID }{m.Projects}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_projects", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkProjects) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_projects WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkProjects: delete: %w", err)
	}
	for _, pid := range m.Projects {
		if err := writeWorkProject(ctx, tx, newID(), m.WorkID, pid, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkProjects: %w", err)
		}
	}
	return nil
}

type DeleteWorkProjects struct{ WorkID ID }

func (m *DeleteWorkProjects) mutationName() string { return "DeleteWorkProjects" }
func (m *DeleteWorkProjects) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkProjects) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_projects", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkProjects) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_projects WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkProjects: %w", err)
	}
	return nil
}

// --- SetWorkOrganizations / DeleteWorkOrganizations ---

type SetWorkOrganizations struct {
	WorkID        ID
	Organizations []ID
	userID        *ID
}

func (m *SetWorkOrganizations) mutationName() string { return "SetWorkOrganizations" }
func (m *SetWorkOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkOrganizations) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: struct{ Organizations []ID }{m.Organizations}},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_organizations", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_organizations WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkOrganizations: delete: %w", err)
	}
	for _, oid := range m.Organizations {
		if err := writeWorkOrganization(ctx, tx, newID(), m.WorkID, oid, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkOrganizations: %w", err)
		}
	}
	return nil
}

type DeleteWorkOrganizations struct{ WorkID ID }

func (m *DeleteWorkOrganizations) mutationName() string { return "DeleteWorkOrganizations" }
func (m *DeleteWorkOrganizations) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkOrganizations) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_organizations", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkOrganizations) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_organizations WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkOrganizations: %w", err)
	}
	return nil
}

// --- SetWorkRels / DeleteWorkRels ---

type SetWorkRels struct {
	WorkID ID
	Rels   []struct {
		RelatedWorkID ID
		Kind          string
	}
	userID *ID
}

func (m *SetWorkRels) mutationName() string { return "SetWorkRels" }
func (m *SetWorkRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkRels) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	m.userID = in.UserID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpUpdate,
		diff:       Diff{Args: m.Rels},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_rels", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *SetWorkRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_rels WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("SetWorkRels: delete: %w", err)
	}
	for _, r := range m.Rels {
		if err := writeWorkRel(ctx, tx, newID(), m.WorkID, r.RelatedWorkID, r.Kind, nil, m.userID); err != nil {
			return fmt.Errorf("SetWorkRels: %w", err)
		}
	}
	return nil
}

type DeleteWorkRels struct{ WorkID ID }

func (m *DeleteWorkRels) mutationName() string { return "DeleteWorkRels" }
func (m *DeleteWorkRels) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkRels) apply(state mutationState, in AddRevInput) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   m.WorkID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinCollective(ctx, tx, "bbl_work_rels", "work_id", m.WorkID, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}
func (m *DeleteWorkRels) write(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_rels WHERE work_id = $1 AND user_id IS NOT NULL`, m.WorkID); err != nil {
		return fmt.Errorf("DeleteWorkRels: %w", err)
	}
	return nil
}
