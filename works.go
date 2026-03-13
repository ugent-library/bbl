package bbl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
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

	revID := newID()
	if _, err := tx.Exec(ctx, `
		INSERT INTO bbl_revs (id, source) VALUES ($1, $2)`,
		revID, source); err != nil {
		return 0, fmt.Errorf("importWorkBatch: %w", err)
	}

	var changedWorkIDs []ID
	var n int
	for _, in := range records {
		workID, isNew, err := r.importWorkRecord(ctx, tx, source, in, revID, priorities)
		if err != nil {
			return n, fmt.Errorf("importWorkBatch: source_id=%s: %w", in.SourceID, err)
		}
		changedWorkIDs = append(changedWorkIDs, workID)
		_ = isNew
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

func (r *Repo) importWorkRecord(ctx context.Context, tx pgx.Tx, source string, in *ImportWorkInput, revID ID, priorities map[string]int) (ID, bool, error) {
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
		return ID{}, false, err
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
			return ID{}, false, fmt.Errorf("insert bbl_works: %w", err)
		}
		sourceRecordID = newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_sources (id, work_id, source, source_id, record, ingested_at)
			VALUES ($1, $2, $3, $4, $5, transaction_timestamp())`,
			sourceRecordID, workID, source, in.SourceID, in.SourceRecord); err != nil {
			return ID{}, false, fmt.Errorf("insert bbl_work_sources: %w", err)
		}
	} else {
		if err := deleteSourceAssertions(ctx, tx, workAssertionTables, "work_source_id", sourceRecordID); err != nil {
			return ID{}, false, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_work_sources SET record = $1, ingested_at = transaction_timestamp()
			WHERE id = $2`,
			in.SourceRecord, sourceRecordID); err != nil {
			return ID{}, false, fmt.Errorf("update bbl_work_sources: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE bbl_works SET version = version + 1, updated_at = transaction_timestamp()
			WHERE id = $1`, workID); err != nil {
			return ID{}, false, fmt.Errorf("bump version: %w", err)
		}
	}

	// Insert scalar field assertions.
	if err := importWorkFields(ctx, tx, workID, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Insert relation assertions.
	if err := importWorkRelations(ctx, tx, workID, source, sourceRecordID, in); err != nil {
		return ID{}, false, err
	}

	// Auto-pin all grouping keys.
	if err := autoPinAllWork(ctx, tx, workID, priorities); err != nil {
		return ID{}, false, err
	}

	// Audit row.
	opType := OpUpdate
	if isNew {
		opType = OpCreate
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO bbl_mutations (rev_id, name, entity_type, entity_id, op_type, diff)
		VALUES ($1, $2, $3, $4, $5, '{}')`,
		revID, "ImportWork", RecordTypeWork, workID, opType); err != nil {
		return ID{}, false, fmt.Errorf("insert bbl_mutations: %w", err)
	}

	return workID, isNew, nil
}

// importWorkFields inserts scalar assertion rows for non-empty fields.
func importWorkFields(ctx context.Context, tx pgx.Tx, workID ID, sourceRecordID ID, in *ImportWorkInput) error {
	type sf struct {
		field string
		val   string
	}
	for _, f := range []sf{
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
		if err := writeCreateWorkField(ctx, tx, newID(), workID, f.field, f.val, &sourceRecordID, nil); err != nil {
			return err
		}
	}
	if in.Conference != (Conference{}) {
		if err := writeCreateWorkField(ctx, tx, newID(), workID, "conference", in.Conference, &sourceRecordID, nil); err != nil {
			return err
		}
	}
	if in.Pages != (Extent{}) {
		if err := writeCreateWorkField(ctx, tx, newID(), workID, "pages", in.Pages, &sourceRecordID, nil); err != nil {
			return err
		}
	}
	return nil
}

// importWorkRelations inserts relation assertion rows.
func importWorkRelations(ctx context.Context, tx pgx.Tx, workID ID, source string, sourceRecordID ID, in *ImportWorkInput) error {
	for _, id := range in.Identifiers {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_identifiers (id, work_id, scheme, val, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, id.Scheme, id.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_identifiers: %w", err)
		}
	}
	for _, cl := range in.Classifications {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_classifications (id, work_id, scheme, val, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, cl.Scheme, cl.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_classifications: %w", err)
		}
	}
	for i, c := range in.Contributors {
		var personID *ID
		if c.PersonRef != nil {
			pid, err := resolvePersonRef(ctx, tx, *c.PersonRef, source)
			if err == nil {
				personID = &pid
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_contributors
			    (id, work_id, position, person_id, name, given_name, family_name, roles, work_source_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			newID(), workID, fracdexPos(i), personID,
			nilIfEmpty(c.Name), nilIfEmpty(c.GivenName), nilIfEmpty(c.FamilyName),
			c.Roles, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_contributors: %w", err)
		}
	}
	for _, t := range in.Titles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_titles (id, work_id, lang, val, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, t.Lang, t.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_titles: %w", err)
		}
	}
	for _, a := range in.Abstracts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_abstracts (id, work_id, lang, val, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, a.Lang, a.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_abstracts: %w", err)
		}
	}
	for _, ls := range in.LaySummaries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_lay_summaries (id, work_id, lang, val, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, ls.Lang, ls.Val, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_lay_summaries: %w", err)
		}
	}
	for _, n := range in.Notes {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_notes (id, work_id, val, kind, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, n.Val, nilIfEmpty(n.Kind), sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_notes: %w", err)
		}
	}
	for _, kw := range in.Keywords {
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_keywords (id, work_id, val, work_source_id)
			VALUES ($1, $2, $3, $4)`,
			newID(), workID, kw, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_keywords: %w", err)
		}
	}
	for _, p := range in.Projects {
		projectID, err := resolveProjectRef(ctx, tx, p.Ref, source)
		if err != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_projects (id, work_id, project_id, work_source_id)
			VALUES ($1, $2, $3, $4)`,
			newID(), workID, projectID, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_projects: %w", err)
		}
	}
	for _, o := range in.Organizations {
		orgID, err := resolveOrganizationRef(ctx, tx, o.Ref, source)
		if err != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_organizations (id, work_id, organization_id, work_source_id)
			VALUES ($1, $2, $3, $4)`,
			newID(), workID, orgID, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_organizations: %w", err)
		}
	}
	for _, rel := range in.RelatedWorks {
		relWorkID, err := resolveWorkRef(ctx, tx, rel.Ref, source)
		if err != nil {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bbl_work_rels (id, work_id, related_work_id, kind, work_source_id)
			VALUES ($1, $2, $3, $4, $5)`,
			newID(), workID, relWorkID, rel.Kind, sourceRecordID); err != nil {
			return fmt.Errorf("insert bbl_work_rels: %w", err)
		}
	}
	return nil
}

// fracdexPos returns a fracdex position string for an initial ordered insert.
// For the first bulk insert, simple zero-padded integers work fine.
func fracdexPos(i int) string {
	return fmt.Sprintf("%06d", i)
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
		StrFields       []struct {
			Field string          `json:"field"`
			Val   json.RawMessage `json:"val"`
		} `json:"str_fields,omitempty"`
		Identifiers     []WorkIdentifier     `json:"identifiers,omitempty"`
		Classifications []WorkClassification `json:"classifications,omitempty"`
		Contributors    []WorkContributor    `json:"contributors,omitempty"`
		Titles          []Title              `json:"titles,omitempty"`
		Abstracts       []Text               `json:"abstracts,omitempty"`
		LaySummaries    []Text               `json:"lay_summaries,omitempty"`
		Notes           []Note               `json:"notes,omitempty"`
		Keywords        []Keyword            `json:"keywords,omitempty"`
	}
	if err := json.Unmarshal(cache, &d); err != nil {
		return fmt.Errorf("parseWorkCache: %w", err)
	}
	for _, sf := range d.StrFields {
		setWorkField(w, sf.Field, sf.Val)
	}
	w.Identifiers = d.Identifiers
	w.Classifications = d.Classifications
	w.Contributors = d.Contributors
	w.Titles = d.Titles
	w.Abstracts = d.Abstracts
	w.LaySummaries = d.LaySummaries
	w.Notes = d.Notes
	w.Keywords = d.Keywords
	return nil
}

// setWorkField sets a scalar field on Work from a JSON-encoded value.
func setWorkField(w *Work, field string, val json.RawMessage) {
	var s string
	switch field {
	case "conference":
		json.Unmarshal(val, &w.Conference)
		return
	case "pages":
		json.Unmarshal(val, &w.Pages)
		return
	}
	if json.Unmarshal(val, &s) != nil {
		return
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
}
