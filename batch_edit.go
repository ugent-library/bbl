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

		for field, ft := range csvFieldTypes("work") {
			csvCols := ft.csv.columns(field)

			changed := false
			for _, col := range csvCols {
				if row.values[col] != current[col] {
					changed = true
					break
				}
			}
			if !changed {
				result.Skipped++
				continue
			}

			if fieldRev, ok := fieldRevs[field]; ok && fieldRev > row.revID {
				result.Conflicts = append(result.Conflicts, BatchConflict{
					WorkID:     row.workID,
					Field:      field,
					CurrentVal: current[csvCols[0]],
					CSVVal:     row.values[csvCols[0]],
				})
				continue
			}

			val, hasData := ft.csv.unflatten(field, row.values)
			if hasData {
				result.updates = append(result.updates, &Set{RecordType: "work", RecordID: row.workID, Field: field, Val: val})
			} else {
				result.updates = append(result.updates, &Hide{RecordType: "work", RecordID: row.workID, Field: field})
			}
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

// csvFieldTypes returns field name → fieldType for all CSV-enabled fields
// of the given entity type.
func csvFieldTypes(entityType string) map[string]*fieldType {
	fieldTypes, ok := entityFieldTypes[entityType]
	if !ok {
		return nil
	}
	out := make(map[string]*fieldType)
	for field, ftName := range fieldTypes {
		ft := fieldTypeRegistry[ftName]
		if ft != nil && ft.csv != nil {
			out[field] = ft
		}
	}
	return out
}

// scalarBatchColumns returns all CSV columns for CSV-enabled fields.
func scalarBatchColumns() []string {
	var cols []string
	for field, ft := range csvFieldTypes("work") {
		cols = append(cols, ft.csv.columns(field)...)
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

// flattenScalarField uses the fieldType's CSV codec to flatten a DB value
// into CSV column→value pairs.
func flattenScalarField(fields map[string]string, field string, val json.RawMessage) {
	ft, err := resolveFieldType("work", field)
	if err != nil || ft.csv == nil {
		return
	}
	goVal, err := ft.unmarshal(val)
	if err != nil {
		return
	}
	for k, v := range ft.csv.flatten(field, goVal) {
		fields[k] = v
	}
}
