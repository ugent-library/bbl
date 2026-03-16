package app

import (
	"context"
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/sru"
)

func (app *App) sruWorksHandler() http.Handler {
	enc, _ := bbl.NewWorkEncoder("dc")

	return sru.Handler(sru.ServerConfig{
		Database: "works",
		Title:    "Work records",
		Indexes: []sru.Index{
			{CQLName: "cql.serverChoice", Title: "Free text"},
		},
		Schemas: []sru.Schema{
			{Name: "dc", Identifier: sru.SchemaDC, Title: "Dublin Core"},
		},
		Search: func(ctx context.Context, index, value string, offset, size int) (*sru.SearchResult, error) {
			opts := &bbl.SearchOpts{
				Query:  value,
				Size:   size,
				Offset: offset,
			}
			hits, err := app.services.SearchPublicWorkRecords(ctx, opts)
			if err != nil {
				return nil, err
			}
			var records [][]byte
			for _, h := range hits.Hits {
				data, err := enc.Encode(h.Work)
				if err != nil {
					continue
				}
				records = append(records, data)
			}
			return &sru.SearchResult{Total: hits.Total, Records: records}, nil
		},
	})
}
