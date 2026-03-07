package app

import (
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/bind"
)

func bindSearchOpts(r *http.Request, facets []string) (*bbl.SearchOpts, error) {
	opts := &bbl.SearchOpts{
		Size:   20,
		Facets: []string{"kind"},
	}

	var f string

	b := bind.Request(r).
		Form().
		Vacuum().
		String("q", &opts.Query).
		String("f", &f).
		Int("size", &opts.Size).
		Int("from", &opts.From).
		String("cursor", &opts.Cursor)
	if err := b.Err(); err != nil {
		return nil, err
	}

	if f != "" {
		queryFilter, err := bbl.ParseQueryFilter(f)
		if err != nil {
			return nil, err
		}
		opts.QueryFilter = queryFilter
	}

	for _, field := range opts.Facets {
		if b.Has(field) {
			opts.AddTermsFilter(field, b.GetAll(field)...)
		}
	}

	return opts, nil
}
