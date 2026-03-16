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


// Discovery handlers — status=public only.

func (app *App) searchWorks(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := bbl.SearchPublicWorks(r.Context(), app.services.Index.Works(), opts)
	if err != nil {
		return err
	}
	return views.SearchWorks(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) searchPeople(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.People().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchPeople(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) searchProjects(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := bbl.SearchPublicProjects(r.Context(), app.services.Index.Projects(), opts)
	if err != nil {
		return err
	}
	return views.SearchProjects(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) searchOrganizations(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.Organizations().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.SearchOrganizations(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

// Backoffice handlers — exclude deleted.

func (app *App) backofficeSearchWorks(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	opts.WithFilter("status", "public", "private")
	hits, err := app.services.Index.Works().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchWorks(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchPeople(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.People().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchPeople(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchProjects(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	opts.WithFilter("status", "public", "private")
	hits, err := app.services.Index.Projects().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchProjects(c.ViewCtx, hits, opts).Render(r.Context(), w)
}

func (app *App) backofficeSearchOrganizations(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	opts := parseSearchOpts(r)
	hits, err := app.services.Index.Organizations().Search(r.Context(), opts)
	if err != nil {
		return err
	}
	return views.BackofficeSearchOrganizations(c.ViewCtx, hits, opts).Render(r.Context(), w)
}
