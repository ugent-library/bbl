package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/dcformat"
	"github.com/ugent-library/bbl/oaipmh"
)

func (app *App) oaiHandler() http.Handler {
	p, _ := oaipmh.NewProvider(oaipmh.Config{
		RepositoryName:  "bbl",
		BaseURL:         app.rootURL + "/oai",
		AdminEmails:     []string{},
		MetadataFormats: []oaipmh.MetadataFormat{oaipmh.OAIDC},
		DeletedRecord:   "no",
		RecordProvider:  &oaiBackend{services: app.services, encoder: &dcformat.OAIWorkEncoder{}},
	})
	return p
}

type oaiBackend struct {
	services *bbl.Services
	encoder  bbl.WorkEncoder
}

// oaiCursor wraps the repo cursor with from/until so resumption tokens are self-contained.
type oaiCursor struct {
	Cursor string    `json:"c,omitempty"`
	From   time.Time `json:"f,omitempty"`
	Until  time.Time `json:"t,omitempty"`
}

func encodeOAICursor(c oaiCursor) string {
	b, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(b)
}

func decodeOAICursor(s string) (oaiCursor, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return oaiCursor{}, err
	}
	var c oaiCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return oaiCursor{}, err
	}
	return c, nil
}

func (b *oaiBackend) GetEarliestDatestamp(ctx context.Context) (time.Time, error) {
	return b.services.Repo.GetEarliestWorkTimestamp(ctx)
}

func (b *oaiBackend) ListRecords(ctx context.Context, q oaipmh.Query) (*oaipmh.Page, error) {
	res, from, until, err := b.listWorks(ctx, q)
	if err != nil {
		return nil, err
	}

	records := make([]*oaipmh.Record, len(res.Works))
	for i, w := range res.Works {
		data, err := b.encoder.Encode(w)
		if err != nil {
			return nil, fmt.Errorf("oaiBackend encode: %w", err)
		}
		records[i] = &oaipmh.Record{
			Header:   workHeader(w),
			Metadata: &oaipmh.Payload{XML: string(data)},
		}
	}

	return &oaipmh.Page{
		Records: records,
		Cursor:  b.wrapCursor(res.Cursor, from, until),
	}, nil
}

func (b *oaiBackend) ListIdentifiers(ctx context.Context, q oaipmh.Query) (*oaipmh.IdentifierPage, error) {
	res, from, until, err := b.listWorks(ctx, q)
	if err != nil {
		return nil, err
	}

	headers := make([]*oaipmh.Header, len(res.Works))
	for i, w := range res.Works {
		headers[i] = workHeader(w)
	}

	return &oaipmh.IdentifierPage{
		Headers: headers,
		Cursor:  b.wrapCursor(res.Cursor, from, until),
	}, nil
}

func (b *oaiBackend) listWorks(ctx context.Context, q oaipmh.Query) (*bbl.ListPublicWorksResult, time.Time, time.Time, error) {
	from, until := q.From, q.Until
	var repoCursor string

	if q.Cursor != "" {
		cur, err := decodeOAICursor(q.Cursor)
		if err != nil {
			return nil, time.Time{}, time.Time{}, oaipmh.ErrBadResumptionToken
		}
		repoCursor = cur.Cursor
		from, until = cur.From, cur.Until
	}

	res, err := b.services.Repo.ListPublicWorks(ctx, bbl.ListPublicWorksOpts{
		From:   from,
		Until:  until,
		Cursor: repoCursor,
		Limit:  q.Limit,
	})
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	return res, from, until, nil
}

func (b *oaiBackend) wrapCursor(repoCursor string, from, until time.Time) string {
	if repoCursor == "" {
		return ""
	}
	return encodeOAICursor(oaiCursor{Cursor: repoCursor, From: from, Until: until})
}

func (b *oaiBackend) GetRecord(ctx context.Context, id, metadataPrefix string) (*oaipmh.Record, error) {
	workID, err := bbl.ParseID(id)
	if err != nil {
		return nil, oaipmh.ErrIDDoesNotExist
	}
	w, err := b.services.Repo.GetWork(ctx, workID)
	if err == bbl.ErrNotFound {
		return nil, oaipmh.ErrIDDoesNotExist
	}
	if err != nil {
		return nil, err
	}
	if w.Status != "public" {
		return nil, oaipmh.ErrIDDoesNotExist
	}
	data, err := b.encoder.Encode(w)
	if err != nil {
		return nil, fmt.Errorf("oaiBackend encode: %w", err)
	}
	return &oaipmh.Record{
		Header:   workHeader(w),
		Metadata: &oaipmh.Payload{XML: string(data)},
	}, nil
}

func workHeader(w *bbl.Work) *oaipmh.Header {
	return &oaipmh.Header{
		Identifier: w.ID.String(),
		Datestamp:  w.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
