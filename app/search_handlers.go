package app

import (
	"net/http"
	"strconv"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
)

const defaultPageSize = 20

func parseSearchOpts(r *http.Request) *bbl.SearchOpts {
	opts := &bbl.SearchOpts{
		Query: r.URL.Query().Get("q"),
		Size:  defaultPageSize,
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && v > 0 {
		opts.Offset = v
	}
	return opts
}

func addFilter(opts *bbl.SearchOpts, field string, terms ...string) {
	cond := &bbl.AndCondition{Terms: &bbl.TermsFilter{Field: field, Terms: terms}}
	if opts.Filter == nil {
		opts.Filter = &bbl.QueryFilter{}
	}
	opts.Filter.And = append(opts.Filter.And, cond)
}

// Discovery handlers — status=public only.

func (app *App) searchWorks(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	addFilter(opts, "status", "public")
	hits, err := app.services.Index.Works().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchWorks(hits, opts).Render(r.Context(), w)
}

func (app *App) searchPeople(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.People().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchPeople(hits, opts).Render(r.Context(), w)
}

func (app *App) searchProjects(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	addFilter(opts, "status", "public")
	hits, err := app.services.Index.Projects().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchProjects(hits, opts).Render(r.Context(), w)
}

func (app *App) searchOrganizations(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.Organizations().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchOrganizations(hits, opts).Render(r.Context(), w)
}

// Backoffice handlers — exclude deleted.

func (app *App) backofficeSearchWorks(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	addFilter(opts, "status", "public", "private")
	hits, err := app.services.Index.Works().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchWorks(hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchPeople(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.People().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchPeople(hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchProjects(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	addFilter(opts, "status", "public", "private")
	hits, err := app.services.Index.Projects().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchProjects(hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchOrganizations(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.Organizations().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchOrganizations(hits, opts).Render(r.Context(), w)
}
