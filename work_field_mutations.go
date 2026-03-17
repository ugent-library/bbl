package bbl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// --- shared write helpers ---

// writeCreateWorkField inserts a scalar assertion into bbl_work_fields.
// Shared by both Set mutations (human path) and import.
func writeCreateWorkField(ctx context.Context, tx pgx.Tx, id, workID ID, field string, val any, workSourceID *ID, userID *ID) error {
	valJSON, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO bbl_work_fields (id, work_id, field, val, work_source_id, user_id)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, workID, field, valJSON, workSourceID, userID)
	if err != nil {
		return fmt.Errorf("writeCreateWorkField(%s): %w", field, err)
	}
	return nil
}

// --- Set/Delete helpers for scalar fields ---

func applySetWorkField(workID ID, field string, val any, id *ID, mutUserID **ID, userID *ID) (*mutationEffect, error) {
	*id = newID()
	*mutUserID = userID
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		opType:     OpUpdate,
		diff:       Diff{Args: val},
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinScalar(ctx, tx, "bbl_work_fields", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeSetWorkField(ctx context.Context, tx pgx.Tx, id, workID ID, field string, val any, userID *ID) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_work_fields WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		workID, field); err != nil {
		return fmt.Errorf("writeSetWorkField(%s): delete: %w", field, err)
	}
	return writeCreateWorkField(ctx, tx, id, workID, field, val, nil, userID)
}

func applyDeleteWorkField(workID ID, field string) (*mutationEffect, error) {
	return &mutationEffect{
		recordType: RecordTypeWork,
		recordID:   workID,
		opType:     OpDelete,
		autoPin: func(ctx context.Context, tx pgx.Tx, priorities map[string]int) error {
			return autoPinScalar(ctx, tx, "bbl_work_fields", "work_id", workID, field, "work_source_id", "bbl_work_sources", priorities)
		},
	}, nil
}

func writeDeleteWorkField(ctx context.Context, tx pgx.Tx, workID ID, field string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM bbl_work_fields WHERE work_id = $1 AND field = $2 AND user_id IS NOT NULL`,
		workID, field); err != nil {
		return fmt.Errorf("writeDeleteWorkField(%s): %w", field, err)
	}
	return nil
}

// --- SetWorkArticleNumber / DeleteWorkArticleNumber ---

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

type DeleteWorkArticleNumber struct{ WorkID ID }

func (m *DeleteWorkArticleNumber) mutationName() string { return "delete_work_article_number" }
func (m *DeleteWorkArticleNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkArticleNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "article_number")
}
func (m *DeleteWorkArticleNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "article_number")
}

// --- SetWorkBookTitle / DeleteWorkBookTitle ---

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

type DeleteWorkBookTitle struct{ WorkID ID }

func (m *DeleteWorkBookTitle) mutationName() string { return "delete_work_book_title" }
func (m *DeleteWorkBookTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkBookTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "book_title")
}
func (m *DeleteWorkBookTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "book_title")
}

// --- SetWorkConference / DeleteWorkConference ---

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

type DeleteWorkConference struct{ WorkID ID }

func (m *DeleteWorkConference) mutationName() string { return "delete_work_conference" }
func (m *DeleteWorkConference) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkConference) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "conference")
}
func (m *DeleteWorkConference) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "conference")
}

// --- SetWorkEdition / DeleteWorkEdition ---

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

type DeleteWorkEdition struct{ WorkID ID }

func (m *DeleteWorkEdition) mutationName() string { return "delete_work_edition" }
func (m *DeleteWorkEdition) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkEdition) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "edition")
}
func (m *DeleteWorkEdition) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "edition")
}

// --- SetWorkIssue / DeleteWorkIssue ---

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

type DeleteWorkIssue struct{ WorkID ID }

func (m *DeleteWorkIssue) mutationName() string { return "delete_work_issue" }
func (m *DeleteWorkIssue) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkIssue) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "issue")
}
func (m *DeleteWorkIssue) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "issue")
}

// --- SetWorkIssueTitle / DeleteWorkIssueTitle ---

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

type DeleteWorkIssueTitle struct{ WorkID ID }

func (m *DeleteWorkIssueTitle) mutationName() string { return "delete_work_issue_title" }
func (m *DeleteWorkIssueTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkIssueTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "issue_title")
}
func (m *DeleteWorkIssueTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "issue_title")
}

// --- SetWorkJournalAbbreviation / DeleteWorkJournalAbbreviation ---

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

type DeleteWorkJournalAbbreviation struct{ WorkID ID }

func (m *DeleteWorkJournalAbbreviation) mutationName() string { return "delete_work_journal_abbreviation" }
func (m *DeleteWorkJournalAbbreviation) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkJournalAbbreviation) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "journal_abbreviation")
}
func (m *DeleteWorkJournalAbbreviation) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "journal_abbreviation")
}

// --- SetWorkJournalTitle / DeleteWorkJournalTitle ---

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

type DeleteWorkJournalTitle struct{ WorkID ID }

func (m *DeleteWorkJournalTitle) mutationName() string { return "delete_work_journal_title" }
func (m *DeleteWorkJournalTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkJournalTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "journal_title")
}
func (m *DeleteWorkJournalTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "journal_title")
}

// --- SetWorkPages / DeleteWorkPages ---

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

type DeleteWorkPages struct{ WorkID ID }

func (m *DeleteWorkPages) mutationName() string { return "delete_work_pages" }
func (m *DeleteWorkPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "pages")
}
func (m *DeleteWorkPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "pages")
}

// --- SetWorkPlaceOfPublication / DeleteWorkPlaceOfPublication ---

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

type DeleteWorkPlaceOfPublication struct{ WorkID ID }

func (m *DeleteWorkPlaceOfPublication) mutationName() string { return "delete_work_place_of_publication" }
func (m *DeleteWorkPlaceOfPublication) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkPlaceOfPublication) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "place_of_publication")
}
func (m *DeleteWorkPlaceOfPublication) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "place_of_publication")
}

// --- SetWorkPublicationStatus / DeleteWorkPublicationStatus ---

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

type DeleteWorkPublicationStatus struct{ WorkID ID }

func (m *DeleteWorkPublicationStatus) mutationName() string { return "delete_work_publication_status" }
func (m *DeleteWorkPublicationStatus) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkPublicationStatus) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "publication_status")
}
func (m *DeleteWorkPublicationStatus) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "publication_status")
}

// --- SetWorkPublicationYear / DeleteWorkPublicationYear ---

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

type DeleteWorkPublicationYear struct{ WorkID ID }

func (m *DeleteWorkPublicationYear) mutationName() string { return "delete_work_publication_year" }
func (m *DeleteWorkPublicationYear) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkPublicationYear) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "publication_year")
}
func (m *DeleteWorkPublicationYear) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "publication_year")
}

// --- SetWorkPublisher / DeleteWorkPublisher ---

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

type DeleteWorkPublisher struct{ WorkID ID }

func (m *DeleteWorkPublisher) mutationName() string { return "delete_work_publisher" }
func (m *DeleteWorkPublisher) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkPublisher) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "publisher")
}
func (m *DeleteWorkPublisher) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "publisher")
}

// --- SetWorkReportNumber / DeleteWorkReportNumber ---

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

type DeleteWorkReportNumber struct{ WorkID ID }

func (m *DeleteWorkReportNumber) mutationName() string { return "delete_work_report_number" }
func (m *DeleteWorkReportNumber) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkReportNumber) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "report_number")
}
func (m *DeleteWorkReportNumber) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "report_number")
}

// --- SetWorkSeriesTitle / DeleteWorkSeriesTitle ---

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

type DeleteWorkSeriesTitle struct{ WorkID ID }

func (m *DeleteWorkSeriesTitle) mutationName() string { return "delete_work_series_title" }
func (m *DeleteWorkSeriesTitle) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkSeriesTitle) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "series_title")
}
func (m *DeleteWorkSeriesTitle) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "series_title")
}

// --- SetWorkTotalPages / DeleteWorkTotalPages ---

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

type DeleteWorkTotalPages struct{ WorkID ID }

func (m *DeleteWorkTotalPages) mutationName() string { return "delete_work_total_pages" }
func (m *DeleteWorkTotalPages) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkTotalPages) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "total_pages")
}
func (m *DeleteWorkTotalPages) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "total_pages")
}

// --- SetWorkVolume / DeleteWorkVolume ---

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

type DeleteWorkVolume struct{ WorkID ID }

func (m *DeleteWorkVolume) mutationName() string { return "delete_work_volume" }
func (m *DeleteWorkVolume) needs() mutationNeeds  { return mutationNeeds{} }
func (m *DeleteWorkVolume) apply(state mutationState, userID *ID) (*mutationEffect, error) {
	return applyDeleteWorkField(m.WorkID, "volume")
}
func (m *DeleteWorkVolume) write(ctx context.Context, tx pgx.Tx) error {
	return writeDeleteWorkField(ctx, tx, m.WorkID, "volume")
}
