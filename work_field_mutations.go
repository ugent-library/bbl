package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreateWorkField inserts a scalar assertion into bbl_work_assertions.
// Shared by both Set mutations (human path) and import.
func writeCreateWorkField(ctx context.Context, tx pgx.Tx, id, workID ID, field string, val any, workSourceID *ID, userID *ID) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_work_assertions (id, work_id, field, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, field, valJSON, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Hide/Unset helpers for scalar fields ---

func applySetWorkField(workID ID, field string, val any, id *ID, mutUserID **ID, userID *ID) (*mutationEffect, error) {
	*id = newID()
	*mutUserID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		opType:     OpUpdate,
		diff:       Diff{Args: val},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeSetWorkField(ctx context.Context, tx pgx.Tx, id, workID ID, field string, val any, userID *ID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_work_assertions WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		workID, field); err != nil {
		return fmt.Errorf("writeSetWorkField(%s): delete: %w", field, err)
	}
	return writeCreateWorkField(ctx, tx, id, workID, field, val, nil, userID)
}

func applyUnsetWorkField(workID ID, field string) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPin(ctx, tx, "bbl_work_assertions", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeUnsetWorkField(ctx context.Context, tx pgx.Tx, workID ID, field string) error {
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
	id     ID
	userID *ID
}

func (m *SetWorkArticleNumber) mutationName() string { return "set_work_article_number" }
func (m *SetWorkArticleNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkArticleNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "article_number", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkArticleNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "article_number", m.Val, m.userID)
}

type UnsetWorkArticleNumber struct{ WorkID ID }

func (m *UnsetWorkArticleNumber) mutationName() string { return "unset_work_article_number" }
func (m *UnsetWorkArticleNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkArticleNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "article_number")
}
func (m *UnsetWorkArticleNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "article_number")
}

// --- SetWorkBookTitle / UnsetWorkBookTitle ---

type SetWorkBookTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkBookTitle) mutationName() string { return "set_work_book_title" }
func (m *SetWorkBookTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkBookTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "book_title", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkBookTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "book_title", m.Val, m.userID)
}

type UnsetWorkBookTitle struct{ WorkID ID }

func (m *UnsetWorkBookTitle) mutationName() string { return "unset_work_book_title" }
func (m *UnsetWorkBookTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkBookTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "book_title")
}
func (m *UnsetWorkBookTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "book_title")
}

// --- SetWorkConference / UnsetWorkConference ---

type SetWorkConference struct {
	WorkID ID     `json:"work_id"`
	Val    Conference `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkConference) mutationName() string { return "set_work_conference" }
func (m *SetWorkConference) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkConference) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "conference", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkConference) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "conference", m.Val, m.userID)
}

type UnsetWorkConference struct{ WorkID ID }

func (m *UnsetWorkConference) mutationName() string { return "unset_work_conference" }
func (m *UnsetWorkConference) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkConference) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "conference")
}
func (m *UnsetWorkConference) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "conference")
}

// --- SetWorkEdition / UnsetWorkEdition ---

type SetWorkEdition struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkEdition) mutationName() string { return "set_work_edition" }
func (m *SetWorkEdition) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkEdition) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "edition", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkEdition) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "edition", m.Val, m.userID)
}

type UnsetWorkEdition struct{ WorkID ID }

func (m *UnsetWorkEdition) mutationName() string { return "unset_work_edition" }
func (m *UnsetWorkEdition) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkEdition) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "edition")
}
func (m *UnsetWorkEdition) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "edition")
}

// --- SetWorkIssue / UnsetWorkIssue ---

type SetWorkIssue struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkIssue) mutationName() string { return "set_work_issue" }
func (m *SetWorkIssue) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkIssue) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "issue", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkIssue) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "issue", m.Val, m.userID)
}

type UnsetWorkIssue struct{ WorkID ID }

func (m *UnsetWorkIssue) mutationName() string { return "unset_work_issue" }
func (m *UnsetWorkIssue) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkIssue) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "issue")
}
func (m *UnsetWorkIssue) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "issue")
}

// --- SetWorkIssueTitle / UnsetWorkIssueTitle ---

type SetWorkIssueTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkIssueTitle) mutationName() string { return "set_work_issue_title" }
func (m *SetWorkIssueTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkIssueTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "issue_title", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkIssueTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "issue_title", m.Val, m.userID)
}

type UnsetWorkIssueTitle struct{ WorkID ID }

func (m *UnsetWorkIssueTitle) mutationName() string { return "unset_work_issue_title" }
func (m *UnsetWorkIssueTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkIssueTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "issue_title")
}
func (m *UnsetWorkIssueTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "issue_title")
}

// --- SetWorkJournalAbbreviation / UnsetWorkJournalAbbreviation ---

type SetWorkJournalAbbreviation struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkJournalAbbreviation) mutationName() string { return "set_work_journal_abbreviation" }
func (m *SetWorkJournalAbbreviation) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkJournalAbbreviation) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "journal_abbreviation", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "journal_abbreviation", m.Val, m.userID)
}

type UnsetWorkJournalAbbreviation struct{ WorkID ID }

func (m *UnsetWorkJournalAbbreviation) mutationName() string { return "unset_work_journal_abbreviation" }
func (m *UnsetWorkJournalAbbreviation) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkJournalAbbreviation) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "journal_abbreviation")
}
func (m *UnsetWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "journal_abbreviation")
}

// --- SetWorkJournalTitle / UnsetWorkJournalTitle ---

type SetWorkJournalTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkJournalTitle) mutationName() string { return "set_work_journal_title" }
func (m *SetWorkJournalTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkJournalTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "journal_title", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkJournalTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "journal_title", m.Val, m.userID)
}

type UnsetWorkJournalTitle struct{ WorkID ID }

func (m *UnsetWorkJournalTitle) mutationName() string { return "unset_work_journal_title" }
func (m *UnsetWorkJournalTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkJournalTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "journal_title")
}
func (m *UnsetWorkJournalTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "journal_title")
}

// --- SetWorkPages / UnsetWorkPages ---

type SetWorkPages struct {
	WorkID ID     `json:"work_id"`
	Val    Extent `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkPages) mutationName() string { return "set_work_pages" }
func (m *SetWorkPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "pages", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "pages", m.Val, m.userID)
}

type UnsetWorkPages struct{ WorkID ID }

func (m *UnsetWorkPages) mutationName() string { return "unset_work_pages" }
func (m *UnsetWorkPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "pages")
}
func (m *UnsetWorkPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "pages")
}

// --- SetWorkPlaceOfPublication / UnsetWorkPlaceOfPublication ---

type SetWorkPlaceOfPublication struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkPlaceOfPublication) mutationName() string { return "set_work_place_of_publication" }
func (m *SetWorkPlaceOfPublication) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkPlaceOfPublication) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "place_of_publication", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "place_of_publication", m.Val, m.userID)
}

type UnsetWorkPlaceOfPublication struct{ WorkID ID }

func (m *UnsetWorkPlaceOfPublication) mutationName() string { return "unset_work_place_of_publication" }
func (m *UnsetWorkPlaceOfPublication) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkPlaceOfPublication) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "place_of_publication")
}
func (m *UnsetWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "place_of_publication")
}

// --- SetWorkPublicationStatus / UnsetWorkPublicationStatus ---

type SetWorkPublicationStatus struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkPublicationStatus) mutationName() string { return "set_work_publication_status" }
func (m *SetWorkPublicationStatus) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkPublicationStatus) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "publication_status", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "publication_status", m.Val, m.userID)
}

type UnsetWorkPublicationStatus struct{ WorkID ID }

func (m *UnsetWorkPublicationStatus) mutationName() string { return "unset_work_publication_status" }
func (m *UnsetWorkPublicationStatus) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkPublicationStatus) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "publication_status")
}
func (m *UnsetWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "publication_status")
}

// --- SetWorkPublicationYear / UnsetWorkPublicationYear ---

type SetWorkPublicationYear struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkPublicationYear) mutationName() string { return "set_work_publication_year" }
func (m *SetWorkPublicationYear) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkPublicationYear) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "publication_year", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkPublicationYear) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "publication_year", m.Val, m.userID)
}

type UnsetWorkPublicationYear struct{ WorkID ID }

func (m *UnsetWorkPublicationYear) mutationName() string { return "unset_work_publication_year" }
func (m *UnsetWorkPublicationYear) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkPublicationYear) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "publication_year")
}
func (m *UnsetWorkPublicationYear) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "publication_year")
}

// --- SetWorkPublisher / UnsetWorkPublisher ---

type SetWorkPublisher struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkPublisher) mutationName() string { return "set_work_publisher" }
func (m *SetWorkPublisher) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkPublisher) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "publisher", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkPublisher) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "publisher", m.Val, m.userID)
}

type UnsetWorkPublisher struct{ WorkID ID }

func (m *UnsetWorkPublisher) mutationName() string { return "unset_work_publisher" }
func (m *UnsetWorkPublisher) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkPublisher) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "publisher")
}
func (m *UnsetWorkPublisher) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "publisher")
}

// --- SetWorkReportNumber / UnsetWorkReportNumber ---

type SetWorkReportNumber struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkReportNumber) mutationName() string { return "set_work_report_number" }
func (m *SetWorkReportNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkReportNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "report_number", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkReportNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "report_number", m.Val, m.userID)
}

type UnsetWorkReportNumber struct{ WorkID ID }

func (m *UnsetWorkReportNumber) mutationName() string { return "unset_work_report_number" }
func (m *UnsetWorkReportNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkReportNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "report_number")
}
func (m *UnsetWorkReportNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "report_number")
}

// --- SetWorkSeriesTitle / UnsetWorkSeriesTitle ---

type SetWorkSeriesTitle struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkSeriesTitle) mutationName() string { return "set_work_series_title" }
func (m *SetWorkSeriesTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkSeriesTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "series_title", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "series_title", m.Val, m.userID)
}

type UnsetWorkSeriesTitle struct{ WorkID ID }

func (m *UnsetWorkSeriesTitle) mutationName() string { return "unset_work_series_title" }
func (m *UnsetWorkSeriesTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkSeriesTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "series_title")
}
func (m *UnsetWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "series_title")
}

// --- SetWorkTotalPages / UnsetWorkTotalPages ---

type SetWorkTotalPages struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkTotalPages) mutationName() string { return "set_work_total_pages" }
func (m *SetWorkTotalPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkTotalPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "total_pages", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkTotalPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "total_pages", m.Val, m.userID)
}

type UnsetWorkTotalPages struct{ WorkID ID }

func (m *UnsetWorkTotalPages) mutationName() string { return "unset_work_total_pages" }
func (m *UnsetWorkTotalPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkTotalPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "total_pages")
}
func (m *UnsetWorkTotalPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "total_pages")
}

// --- SetWorkVolume / UnsetWorkVolume ---

type SetWorkVolume struct {
	WorkID ID     `json:"work_id"`
	Val    string `json:"val"`
	id     ID
	userID *ID
}

func (m *SetWorkVolume) mutationName() string { return "set_work_volume" }
func (m *SetWorkVolume) needs() mutationNeeds  { return mutationNeeds{} }
func (m *SetWorkVolume) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applySetWorkField(m.WorkID, "volume", m.Val, &m.id, &m.userID, userID)
}
func (m *SetWorkVolume) write(ctx context.Context, tx pgx.Tx) error {
	return writeSetWorkField(ctx, tx, m.id, m.WorkID, "volume", m.Val, m.userID)
}

type UnsetWorkVolume struct{ WorkID ID }

func (m *UnsetWorkVolume) mutationName() string { return "unset_work_volume" }
func (m *UnsetWorkVolume) needs() mutationNeeds  { return mutationNeeds{} }
func (m *UnsetWorkVolume) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyUnsetWorkField(m.WorkID, "volume")
}
func (m *UnsetWorkVolume) write(ctx context.Context, tx pgx.Tx) error {
	return writeUnsetWorkField(ctx, tx, m.WorkID, "volume")
}
