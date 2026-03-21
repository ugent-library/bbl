package bbl

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"time"
)

// Services bundles the core runtime dependencies.
type Services struct {
	Repo            *Repo
	Index           Index // nil = no indexing
	UserSources     map[string]UserSource
	WorkIterSources map[string]WorkSourceIter
	WorkGetSources  map[string]WorkSourceGetter
}

// UpdateAndIndex writes a revision to the DB and best-effort indexes affected records.
func (s *Services) UpdateAndIndex(ctx context.Context, user *User, updates ...any) (bool, error) {
	ok, effects, err := s.Repo.Update(ctx, user, updates...)
	if err != nil || !ok {
		return ok, err
	}
	s.indexEffects(ctx, effects)
	return true, nil
}

func (s *Services) indexEffects(ctx context.Context, effects []RevEffect) {
	if s.Index == nil {
		return
	}

	// Group IDs by record type.
	var workIDs, personIDs, projectIDs, orgIDs []ID
	versions := make(map[ID]int)
	for _, e := range effects {
		versions[e.RecordID] = e.Version
		switch e.RecordType {
		case RecordTypeWork:
			workIDs = append(workIDs, e.RecordID)
		case RecordTypePerson:
			personIDs = append(personIDs, e.RecordID)
		case RecordTypeProject:
			projectIDs = append(projectIDs, e.RecordID)
		case RecordTypeOrganization:
			orgIDs = append(orgIDs, e.RecordID)
		}
	}

	// Batch-read and index, skipping stale reads.
	if len(workIDs) > 0 {
		works, err := s.Repo.GetWorks(ctx, workIDs)
		if err != nil {
			slog.Error("indexEffects", "record_type", "work", "err", err)
		} else {
			for _, w := range works {
				if w.Version > versions[w.ID] {
					continue // another write happened, skip
				}
				if err := s.Index.Works().Add(ctx, w); err != nil {
					slog.Error("indexEffects", "record_type", "work", "err", err)
				}
			}
		}
	}
	// TODO: add batch-read + index for people, projects, organizations
	// when GetPeople/GetProjects/GetOrganizations are implemented.
}

// ImportWorksAndIndex imports works and best-effort indexes changed records.
// Uses a timestamp taken before the import to re-fetch changed works from the DB,
// because the in-memory Work during import doesn't have the cache populated.
func (s *Services) ImportWorksAndIndex(ctx context.Context, source string, seq iter.Seq2[*ImportWorkInput, error]) (int, error) {
	before := time.Now()
	n, err := s.Repo.ImportWorks(ctx, source, seq)
	if err != nil || n == 0 {
		return n, err
	}
	indexSince(s, ctx, before, func(ctx context.Context, since time.Time) iter.Seq2[*Work, error] {
		return s.Repo.EachWorkSince(ctx, since)
	}, func(ctx context.Context, w *Work) error {
		return s.Index.Works().Add(ctx, w)
	})
	return n, nil
}

// ImportPeopleAndIndex imports people and best-effort indexes changed records.
func (s *Services) ImportPeopleAndIndex(ctx context.Context, source, authProvider string, seq iter.Seq2[*ImportPersonInput, error]) (int, error) {
	before := time.Now()
	n, err := s.Repo.ImportPeople(ctx, source, seq)
	if err != nil || n == 0 {
		return n, err
	}
	indexSince(s, ctx, before, func(ctx context.Context, since time.Time) iter.Seq2[*Person, error] {
		return s.Repo.EachPersonSince(ctx, since)
	}, func(ctx context.Context, p *Person) error {
		return s.Index.People().Add(ctx, p)
	})
	return n, nil
}

// ImportProjectsAndIndex imports projects and best-effort indexes changed records.
func (s *Services) ImportProjectsAndIndex(ctx context.Context, source string, seq iter.Seq2[*ImportProjectInput, error]) (int, error) {
	before := time.Now()
	n, err := s.Repo.ImportProjects(ctx, source, seq)
	if err != nil || n == 0 {
		return n, err
	}
	indexSince(s, ctx, before, func(ctx context.Context, since time.Time) iter.Seq2[*Project, error] {
		return s.Repo.EachProjectSince(ctx, since)
	}, func(ctx context.Context, p *Project) error {
		return s.Index.Projects().Add(ctx, p)
	})
	return n, nil
}

// ImportOrganizationsAndIndex imports organizations and best-effort indexes changed records.
func (s *Services) ImportOrganizationsAndIndex(ctx context.Context, source string, seq iter.Seq2[*ImportOrganizationInput, error]) (int, error) {
	before := time.Now()
	n, err := s.Repo.ImportOrganizations(ctx, source, seq)
	if err != nil || n == 0 {
		return n, err
	}
	indexSince(s, ctx, before, func(ctx context.Context, since time.Time) iter.Seq2[*Organization, error] {
		return s.Repo.EachOrganizationSince(ctx, since)
	}, func(ctx context.Context, o *Organization) error {
		return s.Index.Organizations().Add(ctx, o)
	})
	return n, nil
}

// SearchPublicWorkRecords searches the index for public works and fetches full records.
func (s *Services) SearchPublicWorkRecords(ctx context.Context, opts *SearchOpts) (*WorkRecordHits, error) {
	return s.SearchWorkRecords(ctx, opts.WithFilter("status", "public"))
}

// SearchWorkRecords searches the index and fetches full work records from the repo.
func (s *Services) SearchWorkRecords(ctx context.Context, opts *SearchOpts) (*WorkRecordHits, error) {
	if s.Index == nil {
		return nil, fmt.Errorf("no search index configured")
	}
	res, err := s.Index.Works().Search(ctx, opts)
	if err != nil {
		return nil, err
	}
	ids := make([]ID, len(res.Hits))
	for i, h := range res.Hits {
		ids[i] = h.ID
	}
	works, err := s.Repo.GetWorks(ctx, ids)
	if err != nil {
		return nil, err
	}
	hits := make([]WorkRecordHit, len(works))
	for i, w := range works {
		hits[i] = WorkRecordHit{Work: w}
	}
	return &WorkRecordHits{
		Hits:   hits,
		Total:  res.Total,
		Cursor: res.Cursor,
		Facets: res.Facets,
	}, nil
}

// SearchAllWorkRecords returns an iterator over full work records matching the query,
// using cursor-based pagination internally. Each page of index hits is batch-fetched
// from the repo.
func (s *Services) SearchAllWorkRecords(ctx context.Context, opts *SearchOpts) iter.Seq2[*Work, error] {
	return func(yield func(*Work, error) bool) {
		if s.Index == nil {
			yield(nil, fmt.Errorf("no search index configured"))
			return
		}
		o := &SearchOpts{
			Query:  opts.Query,
			Filter: opts.Filter,
			Size:   searchAllSize,
		}
		for {
			res, err := s.Index.Works().Search(ctx, o)
			if err != nil {
				yield(nil, err)
				return
			}
			ids := make([]ID, len(res.Hits))
			for i, h := range res.Hits {
				ids[i] = h.ID
			}
			works, err := s.Repo.GetWorks(ctx, ids)
			if err != nil {
				yield(nil, err)
				return
			}
			for _, w := range works {
				if !yield(w, nil) {
					return
				}
			}
			if res.Cursor == "" || len(res.Hits) < o.Size {
				return
			}
			o.Cursor = res.Cursor
		}
	}
}

// indexSince is a generic helper that re-reads entities changed since a timestamp
// and best-effort indexes them. Errors are logged, not returned.
func indexSince[T any](s *Services, ctx context.Context, since time.Time, each func(context.Context, time.Time) iter.Seq2[T, error], add func(context.Context, T) error) {
	if s.Index == nil {
		return
	}
	for entity, err := range each(ctx, since) {
		if err != nil {
			slog.Error("indexSince", "err", err)
			break
		}
		if err := add(ctx, entity); err != nil {
			slog.Error("indexSince", "err", err)
		}
	}
}
