package opensearchindex

import (
	"context"

	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/ugent-library/bbl"
)

var versionType = "external"

// assert we implement bbl.Index
var _ bbl.Index = (*Index)(nil)

type Index struct {
	organizationsIndex *recIndex[*bbl.Organization]
	peopleIndex        *recIndex[*bbl.Person]
	projectsIndex      *recIndex[*bbl.Project]
	worksIndex         *recIndex[*bbl.Work]
}

func New(ctx context.Context, client *opensearchapi.Client) (*Index, error) {
	organizationsIndex, err := newRecIndex(ctx, client, "bbl_organizations", organizationSettings, organizationToDoc, generateOrganizationQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	peopleIndex, err := newRecIndex(ctx, client, "bbl_people", personSettings, personToDoc, generatePersonQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	projectsIndex, err := newRecIndex(ctx, client, "bbl_projects", projectSettings, projectToDoc, generateProjectQuery, nil, nil)
	if err != nil {
		return nil, err
	}
	worksIndex, err := newRecIndex(ctx, client, "bbl_works", workSettings, workToDoc, generateWorkQuery, workTermsFilters, generateWorkAggs)
	if err != nil {
		return nil, err
	}

	return &Index{
		organizationsIndex: organizationsIndex,
		peopleIndex:        peopleIndex,
		projectsIndex:      projectsIndex,
		worksIndex:         worksIndex,
	}, nil
}

func (idx *Index) Organizations() bbl.RecIndex[*bbl.Organization] {
	return idx.organizationsIndex
}

func (idx *Index) People() bbl.RecIndex[*bbl.Person] {
	return idx.peopleIndex
}

func (idx *Index) Projects() bbl.RecIndex[*bbl.Project] {
	return idx.projectsIndex
}

func (idx *Index) Works() bbl.RecIndex[*bbl.Work] {
	return idx.worksIndex
}
