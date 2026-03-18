package bbl

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
)

// BatchConflict is a field that changed since export.
type BatchConflict struct {
	WorkID     ID
	Field      string
	CurrentVal string
	CSVVal     string
}

// BatchResult holds the outcome of reading and diffing a batch edit.
type BatchResult struct {
	updates   []any
	Conflicts []BatchConflict
	Skipped   int
}

// Updates returns the updaters to apply. Conflicted changes are included
// (additive assertions are always safe to apply).
func (r *BatchResult) Updates() []any {
	return r.updates
}

// WriteWorkBatch exports scalar fields for the given works as a
// batch-edit CSV.
func WriteWorkBatch(ctx context.Context, repo *Repo, w io.Writer, workIDs []ID) error {
	cols := scalarBatchColumns()
	sort.Strings(cols)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := append([]string{"work_id", "rev_id", "kind"}, cols...)
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, id := range workIDs {
		fields, revID, kind, err := getWorkPinnedScalars(ctx, repo, id)
		if err != nil {
			return fmt.Errorf("WriteWorkBatch: %w", err)
		}
		row := make([]string, len(header))
		row[0] = id.String()
		row[1] = strconv.FormatInt(revID, 10)
		row[2] = kind
		for i, col := range cols {
			row[i+3] = fields[col]
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

// ReadWorkBatch reads a batch-edit CSV, diffs against current pinned
// values, and returns the result.
func ReadWorkBatch(ctx context.Context, repo *Repo, r io.Reader) (*BatchResult, error) {
	rows, err := parseBatchCSV(r)
	if err != nil {
		return nil, err
	}

	result := &BatchResult{}
	for _, row := range rows {
		current, _, _, err := getWorkPinnedScalars(ctx, repo, row.workID)
		if err != nil {
			return nil, fmt.Errorf("ReadWorkBatch: %w", err)
		}
		fieldRevs, err := getWorkFieldRevIDs(ctx, repo, row.workID)
		if err != nil {
			return nil, fmt.Errorf("ReadWorkBatch: %w", err)
		}

		for col, csvVal := range row.values {
			if csvVal == current[col] {
				result.Skipped++
				continue
			}

			update, err := buildScalarUpdate(row.workID, col, csvVal, row.values)
			if err != nil {
				return nil, fmt.Errorf("ReadWorkBatch: %w", err)
			}
			if update == nil {
				continue
			}

			fieldName := canonicalFieldName(col)
			if fieldRev, ok := fieldRevs[fieldName]; ok && fieldRev > row.revID {
				result.Conflicts = append(result.Conflicts, BatchConflict{
					WorkID:     row.workID,
					Field:      col,
					CurrentVal: current[col],
					CSVVal:     csvVal,
				})
				continue
			}

			result.updates = append(result.updates, update)
		}
	}
	return result, nil
}

// --- private ---

type batchRow struct {
	workID ID
	revID  int64
	values map[string]string
}

func parseBatchCSV(r io.Reader) ([]batchRow, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(header) < 3 || header[0] != "work_id" || header[1] != "rev_id" || header[2] != "kind" {
		return nil, fmt.Errorf("invalid header: expected work_id, rev_id, kind, ...")
	}
	fields := header[3:]

	var rows []batchRow
	lineNum := 1
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum+1, err)
		}
		lineNum++

		if len(record) < 3 {
			return nil, fmt.Errorf("line %d: too few columns", lineNum)
		}
		workID, err := ParseID(record[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid work_id: %w", lineNum, err)
		}
		revID, err := strconv.ParseInt(record[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid rev_id: %w", lineNum, err)
		}

		values := make(map[string]string, len(fields))
		for i, col := range fields {
			if i+3 < len(record) {
				values[col] = record[i+3]
			}
		}
		rows = append(rows, batchRow{workID: workID, revID: revID, values: values})
	}
	return rows, nil
}

func scalarBatchColumns() []string {
	var cols []string
	for field, typ := range workFieldCatalog {
		switch typ {
		case "text", "year":
			cols = append(cols, field)
		case "extent":
			cols = append(cols, field+".start", field+".end")
		case "conference":
			cols = append(cols, field+".name", field+".organizer", field+".location")
		}
	}
	return cols
}

func getWorkPinnedScalars(ctx context.Context, repo *Repo, workID ID) (map[string]string, int64, string, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT field, val, rev_id
		FROM bbl_work_assertions
		WHERE work_id = $1 AND pinned = true AND hidden = false AND val IS NOT NULL`,
		workID)
	if err != nil {
		return nil, 0, "", fmt.Errorf("getWorkPinnedScalars: %w", err)
	}
	defer rows.Close()

	fields := make(map[string]string)
	var maxRevID int64
	for rows.Next() {
		var field string
		var val json.RawMessage
		var revID int64
		if err := rows.Scan(&field, &val, &revID); err != nil {
			return nil, 0, "", err
		}
		if revID > maxRevID {
			maxRevID = revID
		}
		flattenScalarField(fields, field, val)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, "", err
	}

	var kind string
	if err := repo.db.QueryRow(ctx, `SELECT kind FROM bbl_works WHERE id = $1`, workID).Scan(&kind); err != nil {
		return nil, 0, "", err
	}
	return fields, maxRevID, kind, nil
}

func getWorkFieldRevIDs(ctx context.Context, repo *Repo, workID ID) (map[string]int64, error) {
	rows, err := repo.db.Query(ctx, `
		SELECT field, rev_id
		FROM bbl_work_assertions
		WHERE work_id = $1 AND pinned = true AND val IS NOT NULL`,
		workID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	revs := make(map[string]int64)
	for rows.Next() {
		var field string
		var revID int64
		if err := rows.Scan(&field, &revID); err != nil {
			return nil, err
		}
		revs[field] = revID
	}
	return revs, rows.Err()
}

func flattenScalarField(fields map[string]string, field string, val json.RawMessage) {
	switch field {
	case "conference":
		var c Conference
		if json.Unmarshal(val, &c) == nil {
			fields["conference.name"] = c.Name
			fields["conference.organizer"] = c.Organizer
			fields["conference.location"] = c.Location
		}
	case "pages":
		var e Extent
		if json.Unmarshal(val, &e) == nil {
			fields["pages.start"] = e.Start
			fields["pages.end"] = e.End
		}
	default:
		var s string
		if json.Unmarshal(val, &s) == nil {
			fields[field] = s
		}
	}
}

func canonicalFieldName(col string) string {
	switch col {
	case "pages.start", "pages.end":
		return "pages"
	case "conference.name", "conference.organizer", "conference.location":
		return "conference"
	default:
		return col
	}
}

func buildScalarUpdate(workID ID, col, csvVal string, allValues map[string]string) (any, error) {
	switch col {
	case "pages.start":
		start := allValues["pages.start"]
		end := allValues["pages.end"]
		if start == "" && end == "" {
			return &HideWorkPages{WorkID: workID}, nil
		}
		return &SetWorkPages{WorkID: workID, Val: Extent{Start: start, End: end}}, nil
	case "pages.end":
		return nil, nil
	case "conference.name":
		name := allValues["conference.name"]
		org := allValues["conference.organizer"]
		loc := allValues["conference.location"]
		if name == "" && org == "" && loc == "" {
			return &HideWorkConference{WorkID: workID}, nil
		}
		return &SetWorkConference{WorkID: workID, Val: Conference{Name: name, Organizer: org, Location: loc}}, nil
	case "conference.organizer", "conference.location":
		return nil, nil
	}

	if csvVal == "" {
		return buildHideUpdate(workID, col)
	}
	return buildSetUpdate(workID, col, csvVal)
}

func buildSetUpdate(workID ID, field, val string) (any, error) {
	switch field {
	case "article_number":
		return &SetWorkArticleNumber{WorkID: workID, Val: val}, nil
	case "book_title":
		return &SetWorkBookTitle{WorkID: workID, Val: val}, nil
	case "edition":
		return &SetWorkEdition{WorkID: workID, Val: val}, nil
	case "issue":
		return &SetWorkIssue{WorkID: workID, Val: val}, nil
	case "issue_title":
		return &SetWorkIssueTitle{WorkID: workID, Val: val}, nil
	case "journal_abbreviation":
		return &SetWorkJournalAbbreviation{WorkID: workID, Val: val}, nil
	case "journal_title":
		return &SetWorkJournalTitle{WorkID: workID, Val: val}, nil
	case "place_of_publication":
		return &SetWorkPlaceOfPublication{WorkID: workID, Val: val}, nil
	case "publication_status":
		return &SetWorkPublicationStatus{WorkID: workID, Val: val}, nil
	case "publication_year":
		return &SetWorkPublicationYear{WorkID: workID, Val: val}, nil
	case "publisher":
		return &SetWorkPublisher{WorkID: workID, Val: val}, nil
	case "report_number":
		return &SetWorkReportNumber{WorkID: workID, Val: val}, nil
	case "series_title":
		return &SetWorkSeriesTitle{WorkID: workID, Val: val}, nil
	case "total_pages":
		return &SetWorkTotalPages{WorkID: workID, Val: val}, nil
	case "volume":
		return &SetWorkVolume{WorkID: workID, Val: val}, nil
	default:
		return nil, fmt.Errorf("unknown scalar field %q", field)
	}
}

func buildHideUpdate(workID ID, field string) (any, error) {
	switch field {
	case "article_number":
		return &HideWorkArticleNumber{WorkID: workID}, nil
	case "book_title":
		return &HideWorkBookTitle{WorkID: workID}, nil
	case "conference":
		return &HideWorkConference{WorkID: workID}, nil
	case "edition":
		return &HideWorkEdition{WorkID: workID}, nil
	case "issue":
		return &HideWorkIssue{WorkID: workID}, nil
	case "issue_title":
		return &HideWorkIssueTitle{WorkID: workID}, nil
	case "journal_abbreviation":
		return &HideWorkJournalAbbreviation{WorkID: workID}, nil
	case "journal_title":
		return &HideWorkJournalTitle{WorkID: workID}, nil
	case "pages":
		return &HideWorkPages{WorkID: workID}, nil
	case "place_of_publication":
		return &HideWorkPlaceOfPublication{WorkID: workID}, nil
	case "publication_status":
		return &HideWorkPublicationStatus{WorkID: workID}, nil
	case "publication_year":
		return &HideWorkPublicationYear{WorkID: workID}, nil
	case "publisher":
		return &HideWorkPublisher{WorkID: workID}, nil
	case "report_number":
		return &HideWorkReportNumber{WorkID: workID}, nil
	case "series_title":
		return &HideWorkSeriesTitle{WorkID: workID}, nil
	case "total_pages":
		return &HideWorkTotalPages{WorkID: workID}, nil
	case "volume":
		return &HideWorkVolume{WorkID: workID}, nil
	default:
		return nil, fmt.Errorf("unknown scalar field %q", field)
	}
}
