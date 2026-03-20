package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreateWorkField inserts a scalar assertion into bbl_work_assertions.
// Shared by both Set updaters (human path) and import.
func writeCreateWorkField(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, workSourceID *ID, userID *ID, role *string) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, work_source_id, user_id, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		revID, workID, field, valJSON, workSourceID, userID, role)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Hide/Unset helpers for scalar fields ---

func applySetWorkField(state updateState, workID ID, field string, mutUserID **ID, mutRole **string, userID *ID, role string) (*updateEffect, error) {
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[workID], field) {
			return nil, ErrCuratorLock
		}
	}
	*mutUserID = userID
	*mutRole = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeSetWorkField(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, val any, userID *ID, role *string) error {
	if err := logWorkHistory(ctx, tx, workID, field, revID); err != nil {
		return fmt.Errorf("writeSetWorkField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`, workID, field); err != nil {
		return fmt.Errorf("writeSetWorkField(%s): %w", field, err)
	}
	return writeCreateWorkField(ctx, tx, revID, workID, field, val, nil, userID, role)
}

func applyHideWorkField(state updateState, workID ID, field string, mutUserID **ID, mutRole **string, userID *ID, role string) (*updateEffect, error) {
	if fieldHidden(state.workAssertions[workID], field) {
		return nil, nil
	}
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[workID], field) {
			return nil, ErrCuratorLock
		}
	}
	*mutUserID = userID
	*mutRole = &role
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeHideWorkField(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string, userID *ID, role *string) error {
	if err := logWorkHistory(ctx, tx, workID, field, revID); err != nil {
		return fmt.Errorf("writeHideWorkField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`, workID, field); err != nil {
		return fmt.Errorf("writeHideWorkField(%s): %w", field, err)
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO bbl_work_assertions (rev_id, work_id, field, val, hidden, work_source_id, user_id, role)
		VALUES ($1, $2, $3, NULL, true, NULL, $4, $5)`,
		revID, workID, field, userID, role)
	if err != nil {
		return fmt.Errorf("writeHideWorkField(%s): %w", field, err)
	}
	return nil
}

func applyUnsetWorkField(state updateState, role string, workID ID, field string) (*updateEffect, error) {
	if role != "curator" {
		if fieldCuratorLocked(state.workAssertions[workID], field) {
			return nil, ErrCuratorLock
		}
	}
	return &updateEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeUnsetWorkField(ctx context.Context, tx pgx.Tx, revID int64, workID ID, field string) error {
	if err := logWorkHistory(ctx, tx, workID, field, revID); err != nil {
		return fmt.Errorf("writeUnsetWorkField(%s): %w", field, err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		workID, field); err != nil {
		return fmt.Errorf("writeUnsetWorkField(%s): %w", field, err)
	}
	return nil
}

// --- SetWorkArticleNumber / UnsetWorkArticleNumber ---

type SetWorkArticleNumber struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkArticleNumber) name() string       { return "set:work_article_number" }
func (m *SetWorkArticleNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkArticleNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.ArticleNumber == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "article_number", &m.userID, &m.role, userID, role)
}
func (m *SetWorkArticleNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "article_number", m.Val, m.userID, m.role)
}

type UnsetWorkArticleNumber struct{ WorkID ID }

func (m *UnsetWorkArticleNumber) name() string       { return "unset:work_article_number" }
func (m *UnsetWorkArticleNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkArticleNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.ArticleNumber == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "article_number")
}
func (m *UnsetWorkArticleNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "article_number")
}

// --- SetWorkBookTitle / UnsetWorkBookTitle ---

type SetWorkBookTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkBookTitle) name() string       { return "set:work_book_title" }
func (m *SetWorkBookTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkBookTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.BookTitle == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "book_title", &m.userID, &m.role, userID, role)
}
func (m *SetWorkBookTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "book_title", m.Val, m.userID, m.role)
}

type UnsetWorkBookTitle struct{ WorkID ID }

func (m *UnsetWorkBookTitle) name() string       { return "unset:work_book_title" }
func (m *UnsetWorkBookTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkBookTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.BookTitle == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "book_title")
}
func (m *UnsetWorkBookTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "book_title")
}

// --- SetWorkConference / UnsetWorkConference ---

type SetWorkConference struct {
	WorkID ID         `json:"work_id"`
	Val    Conference `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkConference) name() string       { return "set:work_conference" }
func (m *SetWorkConference) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkConference) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Conference == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "conference", &m.userID, &m.role, userID, role)
}
func (m *SetWorkConference) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "conference", m.Val, m.userID, m.role)
}

type UnsetWorkConference struct{ WorkID ID }

func (m *UnsetWorkConference) name() string       { return "unset:work_conference" }
func (m *UnsetWorkConference) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkConference) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Conference == (Conference{}) {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "conference")
}
func (m *UnsetWorkConference) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "conference")
}

// --- SetWorkEdition / UnsetWorkEdition ---

type SetWorkEdition struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkEdition) name() string       { return "set:work_edition" }
func (m *SetWorkEdition) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkEdition) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Edition == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "edition", &m.userID, &m.role, userID, role)
}
func (m *SetWorkEdition) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "edition", m.Val, m.userID, m.role)
}

type UnsetWorkEdition struct{ WorkID ID }

func (m *UnsetWorkEdition) name() string       { return "unset:work_edition" }
func (m *UnsetWorkEdition) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkEdition) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Edition == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "edition")
}
func (m *UnsetWorkEdition) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "edition")
}

// --- SetWorkIssue / UnsetWorkIssue ---

type SetWorkIssue struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkIssue) name() string       { return "set:work_issue" }
func (m *SetWorkIssue) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkIssue) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Issue == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "issue", &m.userID, &m.role, userID, role)
}
func (m *SetWorkIssue) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "issue", m.Val, m.userID, m.role)
}

type UnsetWorkIssue struct{ WorkID ID }

func (m *UnsetWorkIssue) name() string       { return "unset:work_issue" }
func (m *UnsetWorkIssue) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkIssue) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Issue == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "issue")
}
func (m *UnsetWorkIssue) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "issue")
}

// --- SetWorkIssueTitle / UnsetWorkIssueTitle ---

type SetWorkIssueTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkIssueTitle) name() string       { return "set:work_issue_title" }
func (m *SetWorkIssueTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkIssueTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.IssueTitle == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "issue_title", &m.userID, &m.role, userID, role)
}
func (m *SetWorkIssueTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "issue_title", m.Val, m.userID, m.role)
}

type UnsetWorkIssueTitle struct{ WorkID ID }

func (m *UnsetWorkIssueTitle) name() string       { return "unset:work_issue_title" }
func (m *UnsetWorkIssueTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkIssueTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.IssueTitle == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "issue_title")
}
func (m *UnsetWorkIssueTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "issue_title")
}

// --- SetWorkJournalAbbreviation / UnsetWorkJournalAbbreviation ---

type SetWorkJournalAbbreviation struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkJournalAbbreviation) name() string       { return "set:work_journal_abbreviation" }
func (m *SetWorkJournalAbbreviation) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkJournalAbbreviation) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.JournalAbbreviation == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "journal_abbreviation", &m.userID, &m.role, userID, role)
}
func (m *SetWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "journal_abbreviation", m.Val, m.userID, m.role)
}

type UnsetWorkJournalAbbreviation struct{ WorkID ID }

func (m *UnsetWorkJournalAbbreviation) name() string { return "unset:work_journal_abbreviation" }
func (m *UnsetWorkJournalAbbreviation) needs() updateNeeds {
	return updateNeeds{workIDs: []ID{m.WorkID}}
}
func (m *UnsetWorkJournalAbbreviation) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.JournalAbbreviation == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "journal_abbreviation")
}
func (m *UnsetWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "journal_abbreviation")
}

// --- SetWorkJournalTitle / UnsetWorkJournalTitle ---

type SetWorkJournalTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkJournalTitle) name() string       { return "set:work_journal_title" }
func (m *SetWorkJournalTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkJournalTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.JournalTitle == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "journal_title", &m.userID, &m.role, userID, role)
}
func (m *SetWorkJournalTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "journal_title", m.Val, m.userID, m.role)
}

type UnsetWorkJournalTitle struct{ WorkID ID }

func (m *UnsetWorkJournalTitle) name() string       { return "unset:work_journal_title" }
func (m *UnsetWorkJournalTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkJournalTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.JournalTitle == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "journal_title")
}
func (m *UnsetWorkJournalTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "journal_title")
}

// --- SetWorkPages / UnsetWorkPages ---

type SetWorkPages struct {
	WorkID ID     `json:"work_id"`
	Val    Extent `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkPages) name() string       { return "set:work_pages" }
func (m *SetWorkPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Pages == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "pages", &m.userID, &m.role, userID, role)
}
func (m *SetWorkPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "pages", m.Val, m.userID, m.role)
}

type UnsetWorkPages struct{ WorkID ID }

func (m *UnsetWorkPages) name() string       { return "unset:work_pages" }
func (m *UnsetWorkPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Pages == (Extent{}) {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "pages")
}
func (m *UnsetWorkPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "pages")
}

// --- SetWorkPlaceOfPublication / UnsetWorkPlaceOfPublication ---

type SetWorkPlaceOfPublication struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkPlaceOfPublication) name() string       { return "set:work_place_of_publication" }
func (m *SetWorkPlaceOfPublication) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkPlaceOfPublication) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PlaceOfPublication == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "place_of_publication", &m.userID, &m.role, userID, role)
}
func (m *SetWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "place_of_publication", m.Val, m.userID, m.role)
}

type UnsetWorkPlaceOfPublication struct{ WorkID ID }

func (m *UnsetWorkPlaceOfPublication) name() string { return "unset:work_place_of_publication" }
func (m *UnsetWorkPlaceOfPublication) needs() updateNeeds {
	return updateNeeds{workIDs: []ID{m.WorkID}}
}
func (m *UnsetWorkPlaceOfPublication) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PlaceOfPublication == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "place_of_publication")
}
func (m *UnsetWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "place_of_publication")
}

// --- SetWorkPublicationStatus / UnsetWorkPublicationStatus ---

type SetWorkPublicationStatus struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkPublicationStatus) name() string       { return "set:work_publication_status" }
func (m *SetWorkPublicationStatus) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkPublicationStatus) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PublicationStatus == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "publication_status", &m.userID, &m.role, userID, role)
}
func (m *SetWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "publication_status", m.Val, m.userID, m.role)
}

type UnsetWorkPublicationStatus struct{ WorkID ID }

func (m *UnsetWorkPublicationStatus) name() string       { return "unset:work_publication_status" }
func (m *UnsetWorkPublicationStatus) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkPublicationStatus) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PublicationStatus == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "publication_status")
}
func (m *UnsetWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "publication_status")
}

// --- SetWorkPublicationYear / UnsetWorkPublicationYear ---

type SetWorkPublicationYear struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkPublicationYear) name() string       { return "set:work_publication_year" }
func (m *SetWorkPublicationYear) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkPublicationYear) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PublicationYear == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "publication_year", &m.userID, &m.role, userID, role)
}
func (m *SetWorkPublicationYear) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "publication_year", m.Val, m.userID, m.role)
}

type UnsetWorkPublicationYear struct{ WorkID ID }

func (m *UnsetWorkPublicationYear) name() string       { return "unset:work_publication_year" }
func (m *UnsetWorkPublicationYear) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkPublicationYear) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.PublicationYear == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "publication_year")
}
func (m *UnsetWorkPublicationYear) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "publication_year")
}

// --- SetWorkPublisher / UnsetWorkPublisher ---

type SetWorkPublisher struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkPublisher) name() string       { return "set:work_publisher" }
func (m *SetWorkPublisher) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkPublisher) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Publisher == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "publisher", &m.userID, &m.role, userID, role)
}
func (m *SetWorkPublisher) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "publisher", m.Val, m.userID, m.role)
}

type UnsetWorkPublisher struct{ WorkID ID }

func (m *UnsetWorkPublisher) name() string       { return "unset:work_publisher" }
func (m *UnsetWorkPublisher) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkPublisher) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Publisher == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "publisher")
}
func (m *UnsetWorkPublisher) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "publisher")
}

// --- SetWorkReportNumber / UnsetWorkReportNumber ---

type SetWorkReportNumber struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkReportNumber) name() string       { return "set:work_report_number" }
func (m *SetWorkReportNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkReportNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.ReportNumber == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "report_number", &m.userID, &m.role, userID, role)
}
func (m *SetWorkReportNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "report_number", m.Val, m.userID, m.role)
}

type UnsetWorkReportNumber struct{ WorkID ID }

func (m *UnsetWorkReportNumber) name() string       { return "unset:work_report_number" }
func (m *UnsetWorkReportNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkReportNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.ReportNumber == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "report_number")
}
func (m *UnsetWorkReportNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "report_number")
}

// --- SetWorkSeriesTitle / UnsetWorkSeriesTitle ---

type SetWorkSeriesTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkSeriesTitle) name() string       { return "set:work_series_title" }
func (m *SetWorkSeriesTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkSeriesTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.SeriesTitle == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "series_title", &m.userID, &m.role, userID, role)
}
func (m *SetWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "series_title", m.Val, m.userID, m.role)
}

type UnsetWorkSeriesTitle struct{ WorkID ID }

func (m *UnsetWorkSeriesTitle) name() string       { return "unset:work_series_title" }
func (m *UnsetWorkSeriesTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkSeriesTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.SeriesTitle == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "series_title")
}
func (m *UnsetWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "series_title")
}

// --- SetWorkTotalPages / UnsetWorkTotalPages ---

type SetWorkTotalPages struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkTotalPages) name() string       { return "set:work_total_pages" }
func (m *SetWorkTotalPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkTotalPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.TotalPages == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "total_pages", &m.userID, &m.role, userID, role)
}
func (m *SetWorkTotalPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "total_pages", m.Val, m.userID, m.role)
}

type UnsetWorkTotalPages struct{ WorkID ID }

func (m *UnsetWorkTotalPages) name() string       { return "unset:work_total_pages" }
func (m *UnsetWorkTotalPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkTotalPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.TotalPages == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "total_pages")
}
func (m *UnsetWorkTotalPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "total_pages")
}

// --- SetWorkVolume / UnsetWorkVolume ---

type SetWorkVolume struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	userID *ID
	role   *string
}

func (m *SetWorkVolume) name() string       { return "set:work_volume" }
func (m *SetWorkVolume) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *SetWorkVolume) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Volume == m.Val {
		return nil, nil
	}
	return applySetWorkField(state, m.WorkID, "volume", &m.userID, &m.role, userID, role)
}
func (m *SetWorkVolume) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeSetWorkField(ctx, tx, revID, m.WorkID, "volume", m.Val, m.userID, m.role)
}

type UnsetWorkVolume struct{ WorkID ID }

func (m *UnsetWorkVolume) name() string       { return "unset:work_volume" }
func (m *UnsetWorkVolume) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *UnsetWorkVolume) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	if w := state.works[m.WorkID]; w != nil && w.Volume == "" {
		return nil, nil
	}
	return applyUnsetWorkField(state, role, m.WorkID, "volume")
}
func (m *UnsetWorkVolume) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeUnsetWorkField(ctx, tx, revID, m.WorkID, "volume")
}

// ============================================================
// Hide updaters for scalar fields
// ============================================================

type HideWorkArticleNumber struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkArticleNumber) name() string       { return "hide:work_article_number" }
func (m *HideWorkArticleNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkArticleNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "article_number", &m.userID, &m.role, userID, role)
}
func (m *HideWorkArticleNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "article_number", m.userID, m.role)
}

type HideWorkBookTitle struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkBookTitle) name() string       { return "hide:work_book_title" }
func (m *HideWorkBookTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkBookTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "book_title", &m.userID, &m.role, userID, role)
}
func (m *HideWorkBookTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "book_title", m.userID, m.role)
}

type HideWorkConference struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkConference) name() string       { return "hide:work_conference" }
func (m *HideWorkConference) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkConference) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "conference", &m.userID, &m.role, userID, role)
}
func (m *HideWorkConference) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "conference", m.userID, m.role)
}

type HideWorkEdition struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkEdition) name() string       { return "hide:work_edition" }
func (m *HideWorkEdition) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkEdition) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "edition", &m.userID, &m.role, userID, role)
}
func (m *HideWorkEdition) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "edition", m.userID, m.role)
}

type HideWorkIssue struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkIssue) name() string       { return "hide:work_issue" }
func (m *HideWorkIssue) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkIssue) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "issue", &m.userID, &m.role, userID, role)
}
func (m *HideWorkIssue) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "issue", m.userID, m.role)
}

type HideWorkIssueTitle struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkIssueTitle) name() string       { return "hide:work_issue_title" }
func (m *HideWorkIssueTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkIssueTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "issue_title", &m.userID, &m.role, userID, role)
}
func (m *HideWorkIssueTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "issue_title", m.userID, m.role)
}

type HideWorkJournalAbbreviation struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkJournalAbbreviation) name() string { return "hide:work_journal_abbreviation" }
func (m *HideWorkJournalAbbreviation) needs() updateNeeds {
	return updateNeeds{workIDs: []ID{m.WorkID}}
}
func (m *HideWorkJournalAbbreviation) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "journal_abbreviation", &m.userID, &m.role, userID, role)
}
func (m *HideWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "journal_abbreviation", m.userID, m.role)
}

type HideWorkJournalTitle struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkJournalTitle) name() string       { return "hide:work_journal_title" }
func (m *HideWorkJournalTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkJournalTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "journal_title", &m.userID, &m.role, userID, role)
}
func (m *HideWorkJournalTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "journal_title", m.userID, m.role)
}

type HideWorkPages struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkPages) name() string       { return "hide:work_pages" }
func (m *HideWorkPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "pages", &m.userID, &m.role, userID, role)
}
func (m *HideWorkPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "pages", m.userID, m.role)
}

type HideWorkPlaceOfPublication struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkPlaceOfPublication) name() string       { return "hide:work_place_of_publication" }
func (m *HideWorkPlaceOfPublication) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkPlaceOfPublication) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "place_of_publication", &m.userID, &m.role, userID, role)
}
func (m *HideWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "place_of_publication", m.userID, m.role)
}

type HideWorkPublicationStatus struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkPublicationStatus) name() string       { return "hide:work_publication_status" }
func (m *HideWorkPublicationStatus) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkPublicationStatus) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "publication_status", &m.userID, &m.role, userID, role)
}
func (m *HideWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "publication_status", m.userID, m.role)
}

type HideWorkPublicationYear struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkPublicationYear) name() string       { return "hide:work_publication_year" }
func (m *HideWorkPublicationYear) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkPublicationYear) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "publication_year", &m.userID, &m.role, userID, role)
}
func (m *HideWorkPublicationYear) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "publication_year", m.userID, m.role)
}

type HideWorkPublisher struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkPublisher) name() string       { return "hide:work_publisher" }
func (m *HideWorkPublisher) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkPublisher) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "publisher", &m.userID, &m.role, userID, role)
}
func (m *HideWorkPublisher) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "publisher", m.userID, m.role)
}

type HideWorkReportNumber struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkReportNumber) name() string       { return "hide:work_report_number" }
func (m *HideWorkReportNumber) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkReportNumber) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "report_number", &m.userID, &m.role, userID, role)
}
func (m *HideWorkReportNumber) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "report_number", m.userID, m.role)
}

type HideWorkSeriesTitle struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkSeriesTitle) name() string       { return "hide:work_series_title" }
func (m *HideWorkSeriesTitle) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkSeriesTitle) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "series_title", &m.userID, &m.role, userID, role)
}
func (m *HideWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "series_title", m.userID, m.role)
}

type HideWorkTotalPages struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkTotalPages) name() string       { return "hide:work_total_pages" }
func (m *HideWorkTotalPages) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkTotalPages) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "total_pages", &m.userID, &m.role, userID, role)
}
func (m *HideWorkTotalPages) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "total_pages", m.userID, m.role)
}

type HideWorkVolume struct {
	WorkID ID
	userID *ID
	role   *string
}

func (m *HideWorkVolume) name() string       { return "hide:work_volume" }
func (m *HideWorkVolume) needs() updateNeeds { return updateNeeds{workIDs: []ID{m.WorkID}} }
func (m *HideWorkVolume) apply(state updateState, userID *ID, role string) (*updateEffect, error) {
	return applyHideWorkField(state, m.WorkID, "volume", &m.userID, &m.role, userID, role)
}
func (m *HideWorkVolume) write(ctx context.Context, tx pgx.Tx, revID int64) error {
	return writeHideWorkField(ctx, tx, revID, m.WorkID, "volume", m.userID, m.role)
}
