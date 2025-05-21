package oaiservice

import (
	"context"
	"errors"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/oaipmh"
	"github.com/ugent-library/bbl/pgxrepo"
)

var metadataFormats = []*oaipmh.MetadataFormat{
	{
		MetadataPrefix:    "oai_dc",
		Schema:            "http://www.openarchives.org/OAI/2.0/oai_dc.xsd",
		MetadataNamespace: "http://www.openarchives.org/OAI/2.0/oai_dc/",
	},
}

// TODO handle deleted, sets, identifier prefix
type Service struct {
	repo *pgxrepo.Repo
}

func New(repo *pgxrepo.Repo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetEarliestDatestamp(context.Context) (time.Time, error) {
	return time.Time{}, nil //TODO
}

func (s *Service) HasMetadataFormat(_ context.Context, metadataPrefix string) (bool, error) {
	for _, format := range metadataFormats {
		if format.MetadataPrefix == metadataPrefix {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) HasSets(context.Context) (bool, error) {
	return false, nil //TODO
}

func (s *Service) HasSet(context.Context, string) (bool, error) {
	return false, nil // TODO
}

func (s *Service) GetMetadataFormats(context.Context) ([]*oaipmh.MetadataFormat, error) {
	return metadataFormats, nil // TODO
}

func (s *Service) GetRecordMetadataFormats(context.Context, string) ([]*oaipmh.MetadataFormat, error) {
	return metadataFormats, nil // TODO
}

func (s *Service) GetSets(context.Context) ([]*oaipmh.Set, *oaipmh.ResumptionToken, error) {
	return nil, nil, nil // TODO
}

func (s *Service) GetMoreSets(context.Context, string) ([]*oaipmh.Set, *oaipmh.ResumptionToken, error) {
	return nil, nil, nil // TODO
}

func (s *Service) GetIdentifiers(ctx context.Context, metadataPrefix, set string, from, until time.Time) ([]*oaipmh.Header, *oaipmh.ResumptionToken, error) {
	reps, cursor, err := s.repo.GetWorkRepresentations(ctx, bbl.GetWorkRepresentationsOpts{
		Limit:        50,
		Scheme:       metadataPrefix,
		UpdatedAtGTE: from,
		UpdatedAtLTE: until,
	})
	if err != nil {
		return nil, nil, err
	}
	hdrs := make([]*oaipmh.Header, len(reps))
	for i, rep := range reps {
		hdrs[i] = &oaipmh.Header{
			Identifier: rep.WorkID,
			Datestamp:  rep.UpdatedAt.UTC().Format(time.RFC3339),
		}
	}

	// TODO completeListSize
	resumptionToken := &oaipmh.ResumptionToken{Value: cursor}

	return hdrs, resumptionToken, nil
}

func (s *Service) GetMoreIdentifiers(ctx context.Context, cursor string) ([]*oaipmh.Header, *oaipmh.ResumptionToken, error) {
	reps, newCursor, err := s.repo.GetMoreWorkRepresentations(ctx, cursor)
	if err != nil {
		return nil, nil, err
	}
	hdrs := make([]*oaipmh.Header, len(reps))
	for i, rep := range reps {
		hdrs[i] = &oaipmh.Header{
			Identifier: rep.WorkID,
			Datestamp:  rep.UpdatedAt.UTC().Format(time.RFC3339),
		}
	}

	// TODO completeListSize
	resumptionToken := &oaipmh.ResumptionToken{Value: newCursor}

	return hdrs, resumptionToken, nil
}

func (s *Service) HasRecord(ctx context.Context, id string) (bool, error) {
	return s.repo.HasWorkRepresentation(ctx, id, "oai_dc")
}

func (s *Service) GetRecords(ctx context.Context, metadataPrefix, set string, from, until time.Time) ([]*oaipmh.Record, *oaipmh.ResumptionToken, error) {
	reps, cursor, err := s.repo.GetWorkRepresentations(ctx, bbl.GetWorkRepresentationsOpts{
		Limit:        50,
		Scheme:       metadataPrefix,
		UpdatedAtGTE: from,
		UpdatedAtLTE: until,
	})
	if err != nil {
		return nil, nil, err
	}
	recs := make([]*oaipmh.Record, len(reps))
	for i, rep := range reps {
		recs[i] = &oaipmh.Record{
			Header: &oaipmh.Header{
				Identifier: rep.WorkID,
				Datestamp:  rep.UpdatedAt.UTC().Format(time.RFC3339),
			},
			Metadata: &oaipmh.Payload{XML: string(rep.Record)},
		}
	}

	// TODO completeListSize
	resumptionToken := &oaipmh.ResumptionToken{Value: cursor}

	return recs, resumptionToken, nil
}

func (s *Service) GetMoreRecords(ctx context.Context, cursor string) ([]*oaipmh.Record, *oaipmh.ResumptionToken, error) {
	reps, newCursor, err := s.repo.GetMoreWorkRepresentations(ctx, cursor)
	if err != nil {
		return nil, nil, err
	}
	recs := make([]*oaipmh.Record, len(reps))
	for i, rep := range reps {
		recs[i] = &oaipmh.Record{
			Header: &oaipmh.Header{
				Identifier: rep.WorkID,
				Datestamp:  rep.UpdatedAt.UTC().Format(time.RFC3339),
			},
			Metadata: &oaipmh.Payload{XML: string(rep.Record)},
		}
	}

	// TODO completeListSize
	resumptionToken := &oaipmh.ResumptionToken{Value: newCursor}

	return recs, resumptionToken, nil
}

func (s *Service) GetRecord(ctx context.Context, id string, metadataPrefix string) (*oaipmh.Record, error) {
	rep, err := s.repo.GetWorkRepresentation(ctx, id, metadataPrefix)
	if errors.Is(err, bbl.ErrNotFound) {
		return nil, oaipmh.ErrCannotDisseminateFormat
	}
	if err != nil {
		return nil, err
	}

	return &oaipmh.Record{
		Header: &oaipmh.Header{
			Identifier: id,
			Datestamp:  rep.UpdatedAt.UTC().Format(time.RFC3339),
		},
		Metadata: &oaipmh.Payload{XML: string(rep.Record)},
	}, nil
}
