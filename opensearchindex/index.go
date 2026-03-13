package opensearchindex

import (
	"context"
	"iter"
	"time"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/ugent-library/bbl"
)

// Compile-time interface checks.
var (
	_ bbl.Index             = (*Index)(nil)
	_ bbl.WorkIndex         = (*WorkIdx)(nil)
	_ bbl.PersonIndex       = (*PersonIdx)(nil)
	_ bbl.ProjectIndex      = (*ProjectIdx)(nil)
	_ bbl.OrganizationIndex = (*OrganizationIdx)(nil)
)

// Index is the OpenSearch implementation of bbl.Index.
type Index struct {
	works         *WorkIdx
	people        *PersonIdx
	projects      *ProjectIdx
	organizations *OrganizationIdx
}

// Config configures the OpenSearch index.
type Config struct {
	Client *opensearchapi.Client
	OnFail func(ctx context.Context, id string, err error)
}

// New creates a new OpenSearch Index, initializing aliases for all entity types.
func New(ctx context.Context, cfg Config) (*Index, error) {
	aliases := []struct {
		alias    string
		settings string
	}{
		{"bbl_works", workSettings},
		{"bbl_people", personSettings},
		{"bbl_projects", projectSettings},
		{"bbl_organizations", organizationSettings},
	}
	for _, a := range aliases {
		if err := initAlias(ctx, cfg.Client, a.alias, a.settings); err != nil {
			return nil, err
		}
	}

	return &Index{
		works: &WorkIdx{inner: &searchIndex[*bbl.Work, bbl.WorkHit]{
			client:     cfg.Client,
			alias:      "bbl_works",
			settings:   workSettings,
			toDoc:      workToDoc,
			toHit:      workToHit,
			buildQuery: buildWorkQuery,
			facetDefs:  workFacetDefs,
			filterDefs: workFilterDefs,
			onFail:     cfg.OnFail,
		}},
		people: &PersonIdx{inner: &searchIndex[*bbl.Person, bbl.PersonHit]{
			client:     cfg.Client,
			alias:      "bbl_people",
			settings:   personSettings,
			toDoc:      personToDoc,
			toHit:      personToHit,
			buildQuery: buildPersonQuery,
			facetDefs:  personFacetDefs,
			filterDefs: personFilterDefs,
			onFail:     cfg.OnFail,
		}},
		projects: &ProjectIdx{inner: &searchIndex[*bbl.Project, bbl.ProjectHit]{
			client:     cfg.Client,
			alias:      "bbl_projects",
			settings:   projectSettings,
			toDoc:      projectToDoc,
			toHit:      projectToHit,
			buildQuery: buildProjectQuery,
			facetDefs:  projectFacetDefs,
			filterDefs: projectFilterDefs,
			onFail:     cfg.OnFail,
		}},
		organizations: &OrganizationIdx{inner: &searchIndex[*bbl.Organization, bbl.OrganizationHit]{
			client:     cfg.Client,
			alias:      "bbl_organizations",
			settings:   organizationSettings,
			toDoc:      organizationToDoc,
			toHit:      organizationToHit,
			buildQuery: buildOrganizationQuery,
			facetDefs:  organizationFacetDefs,
			filterDefs: organizationFilterDefs,
			onFail:     cfg.OnFail,
		}},
	}, nil
}

func (idx *Index) Works() bbl.WorkIndex                 { return idx.works }
func (idx *Index) People() bbl.PersonIndex              { return idx.people }
func (idx *Index) Projects() bbl.ProjectIndex           { return idx.projects }
func (idx *Index) Organizations() bbl.OrganizationIndex { return idx.organizations }

// WorkIdx implements bbl.WorkIndex using OpenSearch.
type WorkIdx struct {
	inner *searchIndex[*bbl.Work, bbl.WorkHit]
}

func (w *WorkIdx) Add(ctx context.Context, work *bbl.Work) error {
	return w.inner.add(ctx, work)
}

func (w *WorkIdx) Delete(ctx context.Context, id bbl.ID) error {
	return w.inner.delete(ctx, id)
}

func (w *WorkIdx) DeleteAll(ctx context.Context) error {
	return w.inner.deleteAll(ctx)
}

func (w *WorkIdx) Reindex(ctx context.Context, all iter.Seq2[*bbl.Work, error], changed func(since time.Time) iter.Seq2[*bbl.Work, error]) error {
	return w.inner.reindex(ctx, all, changed)
}

func (w *WorkIdx) Search(ctx context.Context, opts *bbl.SearchOpts) (*bbl.WorkHits, error) {
	hits, err := w.inner.search(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &bbl.WorkHits{
		Hits:   hits.Hits,
		Total:  hits.Total,
		Cursor: hits.Cursor,
		Facets: hits.Facets,
	}, nil
}

// PersonIdx implements bbl.PersonIndex using OpenSearch.
type PersonIdx struct {
	inner *searchIndex[*bbl.Person, bbl.PersonHit]
}

func (p *PersonIdx) Add(ctx context.Context, person *bbl.Person) error {
	return p.inner.add(ctx, person)
}

func (p *PersonIdx) Delete(ctx context.Context, id bbl.ID) error {
	return p.inner.delete(ctx, id)
}

func (p *PersonIdx) DeleteAll(ctx context.Context) error {
	return p.inner.deleteAll(ctx)
}

func (p *PersonIdx) Reindex(ctx context.Context, all iter.Seq2[*bbl.Person, error], changed func(since time.Time) iter.Seq2[*bbl.Person, error]) error {
	return p.inner.reindex(ctx, all, changed)
}

func (p *PersonIdx) Search(ctx context.Context, opts *bbl.SearchOpts) (*bbl.PersonHits, error) {
	hits, err := p.inner.search(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &bbl.PersonHits{
		Hits:   hits.Hits,
		Total:  hits.Total,
		Cursor: hits.Cursor,
		Facets: hits.Facets,
	}, nil
}

// ProjectIdx implements bbl.ProjectIndex using OpenSearch.
type ProjectIdx struct {
	inner *searchIndex[*bbl.Project, bbl.ProjectHit]
}

func (p *ProjectIdx) Add(ctx context.Context, project *bbl.Project) error {
	return p.inner.add(ctx, project)
}

func (p *ProjectIdx) Delete(ctx context.Context, id bbl.ID) error {
	return p.inner.delete(ctx, id)
}

func (p *ProjectIdx) DeleteAll(ctx context.Context) error {
	return p.inner.deleteAll(ctx)
}

func (p *ProjectIdx) Reindex(ctx context.Context, all iter.Seq2[*bbl.Project, error], changed func(since time.Time) iter.Seq2[*bbl.Project, error]) error {
	return p.inner.reindex(ctx, all, changed)
}

func (p *ProjectIdx) Search(ctx context.Context, opts *bbl.SearchOpts) (*bbl.ProjectHits, error) {
	hits, err := p.inner.search(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &bbl.ProjectHits{
		Hits:   hits.Hits,
		Total:  hits.Total,
		Cursor: hits.Cursor,
		Facets: hits.Facets,
	}, nil
}

// OrganizationIdx implements bbl.OrganizationIndex using OpenSearch.
type OrganizationIdx struct {
	inner *searchIndex[*bbl.Organization, bbl.OrganizationHit]
}

func (o *OrganizationIdx) Add(ctx context.Context, org *bbl.Organization) error {
	return o.inner.add(ctx, org)
}

func (o *OrganizationIdx) Delete(ctx context.Context, id bbl.ID) error {
	return o.inner.delete(ctx, id)
}

func (o *OrganizationIdx) DeleteAll(ctx context.Context) error {
	return o.inner.deleteAll(ctx)
}

func (o *OrganizationIdx) Reindex(ctx context.Context, all iter.Seq2[*bbl.Organization, error], changed func(since time.Time) iter.Seq2[*bbl.Organization, error]) error {
	return o.inner.reindex(ctx, all, changed)
}

func (o *OrganizationIdx) Search(ctx context.Context, opts *bbl.SearchOpts) (*bbl.OrganizationHits, error) {
	hits, err := o.inner.search(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &bbl.OrganizationHits{
		Hits:   hits.Hits,
		Total:  hits.Total,
		Cursor: hits.Cursor,
		Facets: hits.Facets,
	}, nil
}
