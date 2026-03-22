package bbl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ImportWorks runs a full sweep from seq, importing all records in batches.
// Re-import = delete all of this source's assertions for the entity + insert new ones.
// Returns the number of records that resulted in a create or update.
func (r *Repo) ImportWorks(ctx context.Context, source string, seq iter.Seq2[*ImportWorkInput, error]) (int, error) {
	const batchSize = 250
	var pending []*ImportWorkInput
	var total int

	flush := func() error {
		n, err := r.importWorkBatch(ctx, source, pending)
		total += n
		pending = pending[:0]
		return err
	}

	for in, err := range seq {
		if err != nil {
			return total, fmt.Errorf("ImportWorks: %w", err)
		}
		pending = append(pending, in)
		if len(pending) == batchSize {
			if err := flush(); err != nil {
				return total, err
			}
		}
	}
	if len(pending) > 0 {
		if err := flush(); err != nil {
			return total, err
		}
	}
	return total, nil
}

func (r *Repo) importWorkBatch(ctx context.Context, source string, records []*ImportWorkInput) (int, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("importWorkBatch: %w", err)
	}
	defer tx.Rollback(ctx)

	priorities, err := fetchSourcePriorities(ctx, tx)
	if err != nil {
		return 0, fmt.Errorf("importWorkBatch: %w", err)
	}

	var revID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO bbl_revs (source) VALUES ($1) RETURNING id`,
		source).Scan(&revID); err != nil {
		return 0, fmt.Errorf("importWorkBatch: %w", err)
	}

	var changedWorkIDs []ID
	var n int
	for _, in := range records {
		workID, err := r.importWorkRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("importWorkBatch: source_id=%s: %w", in.SourceID, err)
		}
		changedWorkIDs = append(changedWorkIDs, workID)
		n++
	}

	if err := rebuildWorkCache(ctx, tx, changedWorkIDs); err != nil {
		return n, fmt.Errorf("importWorkBatch: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("importWorkBatch: %w", err)
	}
	return n, nil
}

func (r *Repo) importWorkRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportWorkInput, revID int64, priorities map[string]int) (ID, error) {
	var workID ID
	var sourceRecordID ID
	var isNew bool
	err := tx.QueryRow(ctx, `
		SELECT work_id, id FROM bbl_work_sources
		WHERE source = $1 AND source_id = $2
		FOR UPDATE`, source, in.SourceID).Scan(&workID, &sourceRecordID)
	if errors.Is(err, pgx.ErrNoRows) {
		isNew = true
		if in.ID != nil {
			workID = *in.ID
		} else {
			workID = newID()
		}
	} else if err != nil {
		return ID{}, err
	}

	if isNew {
		status := in.Status
		if status == "" {
			status = WorkStatusPrivate
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_works (id, version, kind, status)
			VALUES ($1, 1, $2, $3)`,
			workID, in.Kind, status); err != nil {
			return ID{}, fmt.Errorf("insert bbl_works: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_sources (id, work_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, workID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, fmt.Errorf("insert bbl_work_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, "bbl_work_assertions", "work_source_id", sourceRecordID); err != nil {
			return ID{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_work_sources SET record = $1, ingested_at = transaction_timestamp()
			WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, fmt.Errorf("update bbl_work_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_works SET version = version + 1, updated_at = transaction_timestamp()
			WHERE id = $1`, workID); err != nil {
			return ID{}, fmt.Errorf("bump version: %w", err)
		}
	}

	// Build assertion rows, validate, write via shared pipeline.
	rows, err := workImportAssertions(ctx, tx, source, workID, sourceRecordID, in)
	if err != nil {
		return ID{}, err
	}
	if defs := r.Profiles.FieldDefs(RecordTypeWork, in.Kind); defs != nil {
		status := in.Status
		if status == "" {
			status = WorkStatusPrivate
		}
		if errs := validateRecord(status, assertionRowFields(rows), defs); errs != nil {
			return ID{}, errs.ToError()
		}
	}
	if err := writeAssertionRows(ctx, tx, &pgx.Batch{}, 0, revID, rows); err != nil {
		return ID{}, err
	}

	// Auto-pin all grouping keys.
	if err := autoPinRecord(ctx, tx, RecordTypeWork, workID, priorities); err != nil {
		return ID{}, err
	}

	return workID, nil
}

// workImportAssertions resolves refs and converts an ImportWorkInput into assertion rows.
func workImportAssertions(ctx context.Context, tx pgx.Tx, source string, workID ID, sourceRecordID ID, in *ImportWorkInput) ([]assertionRow, error) {
	var rows []assertionRow
	src := &sourceRecordID

	// Scalar string fields.
	for _, f := range []struct{ field, val string }{
		{"article_number", in.ArticleNumber},
		{"book_title", in.BookTitle},
		{"edition", in.Edition},
		{"issue", in.Issue},
		{"issue_title", in.IssueTitle},
		{"journal_abbreviation", in.JournalAbbreviation},
		{"journal_title", in.JournalTitle},
		{"place_of_publication", in.PlaceOfPublication},
		{"publication_status", in.PublicationStatus},
		{"publication_year", in.PublicationYear},
		{"publisher", in.Publisher},
		{"report_number", in.ReportNumber},
		{"series_title", in.SeriesTitle},
		{"total_pages", in.TotalPages},
		{"volume", in.Volume},
	} {
		if f.val == "" {
			continue
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: f.field, val: f.val, sourceRecordID: src,
		})
	}

	// Compound scalars.
	if in.Conference != (Conference{}) {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "conference", val: in.Conference, sourceRecordID: src,
		})
	}
	if in.Pages != (Extent{}) {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "pages", val: in.Pages, sourceRecordID: src,
		})
	}

	// Pure-value collections.
	if len(in.Identifiers) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "identifiers", val: in.Identifiers, sourceRecordID: src,
		})
	}
	if len(in.Classifications) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "classifications", val: in.Classifications, sourceRecordID: src,
		})
	}
	if len(in.Titles) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "titles", val: in.Titles, sourceRecordID: src,
		})
	}
	if len(in.Abstracts) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "abstracts", val: in.Abstracts, sourceRecordID: src,
		})
	}
	if len(in.LaySummaries) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "lay_summaries", val: in.LaySummaries, sourceRecordID: src,
		})
	}
	if len(in.Notes) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "notes", val: in.Notes, sourceRecordID: src,
		})
	}
	if len(in.Keywords) > 0 {
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "keywords", val: in.Keywords, sourceRecordID: src,
		})
	}

	// FK-bearing: contributors.
	if len(in.Contributors) > 0 {
		contributors := make([]WorkContributor, 0, len(in.Contributors))
		for _, c := range in.Contributors {
			var personID *ID
			name, givenName, familyName := c.Name, c.GivenName, c.FamilyName
			if c.PersonRef != nil {
				person, err := resolvePersonRef(ctx, tx, *c.PersonRef, source)
				if err != nil {
					return nil, fmt.Errorf("workImportAssertions: resolve person ref: %w", err)
				}
				personID = &person.ID
				if name == "" && givenName == "" && familyName == "" {
					name, givenName, familyName = person.Name, person.GivenName, person.FamilyName
				}
			}
			kind := c.Kind
			if kind == "" {
				kind = "person"
			}
			if name == "" {
				name = strings.TrimSpace(givenName + " " + familyName)
			}
			contributors = append(contributors, WorkContributor{
				Kind: kind, Name: name, GivenName: givenName, FamilyName: familyName,
				PersonID: personID, Roles: c.Roles,
			})
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "contributors", val: contributors, sourceRecordID: src,
		})
	}

	// FK-bearing: projects.
	if len(in.Projects) > 0 {
		projectIDs := make([]ID, 0, len(in.Projects))
		for _, p := range in.Projects {
			project, err := resolveProjectRef(ctx, tx, p.Ref, source)
			if err != nil {
				return nil, fmt.Errorf("workImportAssertions: resolve project ref: %w", err)
			}
			projectIDs = append(projectIDs, project.ID)
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "projects", val: projectIDs, sourceRecordID: src,
		})
	}

	// FK-bearing: organizations.
	if len(in.Organizations) > 0 {
		orgIDs := make([]ID, 0, len(in.Organizations))
		for _, o := range in.Organizations {
			org, err := resolveOrganizationRef(ctx, tx, o.Ref, source)
			if err != nil {
				return nil, fmt.Errorf("workImportAssertions: resolve organization ref: %w", err)
			}
			orgIDs = append(orgIDs, org.ID)
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "organizations", val: orgIDs, sourceRecordID: src,
		})
	}

	// FK-bearing: related works.
	if len(in.RelatedWorks) > 0 {
		rels := make([]WorkRel, 0, len(in.RelatedWorks))
		for _, rel := range in.RelatedWorks {
			relWork, err := resolveWorkRef(ctx, tx, rel.Ref, source)
			if err != nil {
				return nil, fmt.Errorf("workImportAssertions: resolve work ref: %w", err)
			}
			rels = append(rels, WorkRel{RelatedWorkID: relWork.ID, Kind: rel.Kind})
		}
		rows = append(rows, assertionRow{
			recordType: RecordTypeWork, recordID: workID,
			field: "rels", val: rels, sourceRecordID: src,
		})
	}

	return rows, nil
}

// ---------- Query methods ----------

// GetWork fetches a work by primary key. The returned Work includes its cache
// (inlined display data). Returns ErrNotFound if no row exists.
func (r *Repo) GetWork(ctx context.Context, id ID) (*Work, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works
		WHERE id = $1`, id)
	w, err := scanWork(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetWork: %w", err)
	}
	return w, nil
}

// GetWorks fetches multiple works by ID, preserving the input order.
// Missing IDs are silently skipped.
func (r *Repo) GetWorks(ctx context.Context, ids []ID) ([]*Work, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works
		WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("GetWorks: %w", err)
	}
	defer rows.Close()

	byID := make(map[ID]*Work, len(ids))
	for rows.Next() {
		w, err := scanWork(rows)
		if err != nil {
			return nil, fmt.Errorf("GetWorks: %w", err)
		}
		byID[w.ID] = w
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetWorks: %w", err)
	}

	result := make([]*Work, 0, len(ids))
	for _, id := range ids {
		if w, ok := byID[id]; ok {
			result = append(result, w)
		}
	}
	return result, nil
}

// GetWorkByIdentifier fetches the work that owns the given scheme:val identifier.
// Returns ErrNotFound if no match.
func (r *Repo) GetWorkByIdentifier(ctx context.Context, scheme, val string) (*Work, error) {
	row := r.db.QueryRow(ctx, `
		SELECT w.id, w.version, w.created_at, w.updated_at,
		       w.created_by_id, w.updated_by_id,
		       w.kind, w.status, w.review_status, w.delete_kind,
		       w.deleted_at, w.deleted_by_id,
		       w.cache
		FROM bbl_works w
		JOIN bbl_work_identifiers i ON i.work_id = w.id
		WHERE i.scheme = $1 AND i.val = $2`, scheme, val)
	w, err := scanWork(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("GetWorkByIdentifier: %w", err)
	}
	return w, nil
}

// scanWork scans a single work row (including cache) from a QueryRow result.
// The cache column is parsed into the typed relation fields on Work.
func scanWork(row pgx.Row) (*Work, error) {
	var w Work
	var createdByID, updatedByID, deletedByID pgtype.UUID
	var reviewStatus, deleteKind pgtype.Text
	var deletedAt pgtype.Timestamptz
	var cache []byte
	if err := row.Scan(
		&w.ID, &w.Version, &w.CreatedAt, &w.UpdatedAt,
		&createdByID, &updatedByID,
		&w.Kind, &w.Status, &reviewStatus, &deleteKind,
		&deletedAt, &deletedByID,
		&cache,
	); err != nil {
		return nil, err
	}
	if createdByID.Valid {
		id := ID(createdByID.Bytes)
		w.CreatedByID = &id
	}
	if updatedByID.Valid {
		id := ID(updatedByID.Bytes)
		w.UpdatedByID = &id
	}
	if deletedByID.Valid {
		id := ID(deletedByID.Bytes)
		w.DeletedByID = &id
	}
	if reviewStatus.Valid {
		w.ReviewStatus = reviewStatus.String
	}
	if deleteKind.Valid {
		w.DeleteKind = deleteKind.String
	}
	if deletedAt.Valid {
		w.DeletedAt = &deletedAt.Time
	}
	if err := parseWorkCache(&w, cache); err != nil {
		return nil, err
	}
	return &w, nil
}

// WorkCursor is a keyset pagination cursor for ListPublicWorks.
// ListPublicWorksOpts holds parameters for ListPublicWorks.
type ListPublicWorksOpts struct {
	From   time.Time
	Until  time.Time
	Cursor string // opaque, from previous result
	Limit  int
}

// ListPublicWorksResult holds the result of ListPublicWorks.
type ListPublicWorksResult struct {
	Works  []*Work
	Cursor string // empty = last page
}

type workCursor struct {
	UpdatedAt time.Time `json:"u"`
	ID        ID        `json:"i"`
}

// GetEarliestWorkTimestamp returns the earliest updated_at of any public work.
func (r *Repo) GetEarliestWorkTimestamp(ctx context.Context) (time.Time, error) {
	var t time.Time
	err := r.db.QueryRow(ctx, `SELECT COALESCE(MIN(updated_at), NOW()) FROM bbl_works WHERE status = 'public'`).Scan(&t)
	if err != nil {
		return time.Time{}, fmt.Errorf("GetEarliestWorkTimestamp: %w", err)
	}
	return t, nil
}

// ListPublicWorks returns a page of public works ordered by (updated_at, id) for keyset pagination.
func (r *Repo) ListPublicWorks(ctx context.Context, opts ListPublicWorksOpts) (*ListPublicWorksResult, error) {
	query := `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works
		WHERE status = 'public'`
	var args []any
	n := 0

	if !opts.From.IsZero() {
		n++
		query += fmt.Sprintf(` AND updated_at >= $%d`, n)
		args = append(args, opts.From)
	}
	if !opts.Until.IsZero() {
		n++
		query += fmt.Sprintf(` AND updated_at <= $%d`, n)
		args = append(args, opts.Until)
	}
	if opts.Cursor != "" {
		cur, err := decodeWorkCursor(opts.Cursor)
		if err != nil {
			return nil, fmt.Errorf("ListPublicWorks: invalid cursor: %w", err)
		}
		n++
		query += fmt.Sprintf(` AND (updated_at, id) > ($%d`, n)
		args = append(args, cur.UpdatedAt)
		n++
		query += fmt.Sprintf(`, $%d)`, n)
		args = append(args, cur.ID)
	}

	query += ` ORDER BY updated_at, id`
	n++
	query += fmt.Sprintf(` LIMIT $%d`, n)
	args = append(args, opts.Limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListPublicWorks: %w", err)
	}
	defer rows.Close()

	var works []*Work
	for rows.Next() {
		w, err := scanWork(rows)
		if err != nil {
			return nil, fmt.Errorf("ListPublicWorks: %w", err)
		}
		works = append(works, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListPublicWorks: %w", err)
	}
	var cursor string
	if len(works) == opts.Limit {
		last := works[len(works)-1]
		cursor = encodeWorkCursor(workCursor{UpdatedAt: last.UpdatedAt, ID: last.ID})
	}
	return &ListPublicWorksResult{Works: works, Cursor: cursor}, nil
}

func encodeWorkCursor(c workCursor) string {
	b, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(b)
}

func decodeWorkCursor(s string) (workCursor, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return workCursor{}, err
	}
	var c workCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return workCursor{}, err
	}
	return c, nil
}

// EachWork returns an iterator over all works, ordered by id.
func (r *Repo) EachWork(ctx context.Context) iter.Seq2[*Work, error] {
	return r.eachWork(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works
		ORDER BY id`)
}

// EachWorkSince returns an iterator over works updated since the given time, ordered by id.
func (r *Repo) EachWorkSince(ctx context.Context, since time.Time) iter.Seq2[*Work, error] {
	return r.eachWork(ctx, `
		SELECT id, version, created_at, updated_at,
		       created_by_id, updated_by_id,
		       kind, status, review_status, delete_kind,
		       deleted_at, deleted_by_id,
		       cache
		FROM bbl_works
		WHERE updated_at >= $1
		ORDER BY id`, since)
}

func (r *Repo) eachWork(ctx context.Context, query string, args ...any) iter.Seq2[*Work, error] {
	return func(yield func(*Work, error) bool) {
		rows, err := r.db.Query(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("eachWork: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			w, err := scanWork(rows)
			if err != nil {
				yield(nil, fmt.Errorf("eachWork: %w", err))
				return
			}
			if !yield(w, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("eachWork: %w", err))
		}
	}
}

// parseWorkCache parses the cache jsonb column into typed relation fields on Work.
func parseWorkCache(w *Work, cache []byte) error {
	if len(cache) == 0 || string(cache) == "{}" {
		return nil
	}
	var d struct {
		StrFields []struct {
			Field string          `json:"field"`
			Val   json.RawMessage `json:"val"`
		} `json:"str_fields,omitempty"`
		Identifiers     []Identifier `json:"identifiers,omitempty"`
		Classifications []Identifier `json:"classifications,omitempty"`
		Titles          []Title      `json:"titles,omitempty"`
		Abstracts       []Text       `json:"abstracts,omitempty"`
		LaySummaries    []Text       `json:"lay_summaries,omitempty"`
		Notes           []Note       `json:"notes,omitempty"`
		Keywords        []Keyword    `json:"keywords,omitempty"`
		Contributors    []struct {
			Val            json.RawMessage `json:"val"`
			PersonID       *ID             `json:"person_id,omitempty"`
			OrganizationID *ID             `json:"organization_id,omitempty"`
		} `json:"contributors,omitempty"`
		Projects      []ID      `json:"projects,omitempty"`
		Organizations []ID      `json:"organizations,omitempty"`
		Rels          []WorkRel `json:"rels,omitempty"`
	}
	if err := json.Unmarshal(cache, &d); err != nil {
		return fmt.Errorf("parseWorkCache: %w", err)
	}
	for _, sf := range d.StrFields {
		if err := setWorkField(w, sf.Field, sf.Val); err != nil {
			return fmt.Errorf("parseWorkCache: field %s: %w", sf.Field, err)
		}
	}
	w.Identifiers = d.Identifiers
	w.Classifications = d.Classifications
	w.Titles = d.Titles
	w.Abstracts = d.Abstracts
	w.LaySummaries = d.LaySummaries
	w.Notes = d.Notes
	w.Keywords = d.Keywords
	for _, c := range d.Contributors {
		var co WorkContributor
		if c.Val != nil {
			if err := json.Unmarshal(c.Val, &co); err != nil {
				return fmt.Errorf("parseWorkCache: unmarshal contributor: %w", err)
			}
		}
		co.PersonID = c.PersonID
		w.Contributors = append(w.Contributors, co)
	}
	w.Projects = d.Projects
	w.Organizations = d.Organizations
	w.Rels = d.Rels
	return nil
}

// setWorkField sets a scalar field on Work from a JSON-encoded value.
func setWorkField(w *Work, field string, val json.RawMessage) error {
	switch field {
	case "conference":
		return json.Unmarshal(val, &w.Conference)
	case "pages":
		return json.Unmarshal(val, &w.Pages)
	}
	var s string
	if err := json.Unmarshal(val, &s); err != nil {
		return err
	}
	switch field {
	case "article_number":
		w.ArticleNumber = s
	case "book_title":
		w.BookTitle = s
	case "edition":
		w.Edition = s
	case "issue":
		w.Issue = s
	case "issue_title":
		w.IssueTitle = s
	case "journal_abbreviation":
		w.JournalAbbreviation = s
	case "journal_title":
		w.JournalTitle = s
	case "place_of_publication":
		w.PlaceOfPublication = s
	case "publication_status":
		w.PublicationStatus = s
	case "publication_year":
		w.PublicationYear = s
	case "publisher":
		w.Publisher = s
	case "report_number":
		w.ReportNumber = s
	case "series_title":
		w.SeriesTitle = s
	case "total_pages":
		w.TotalPages = s
	case "volume":
		w.Volume = s
	}
	return nil
}
