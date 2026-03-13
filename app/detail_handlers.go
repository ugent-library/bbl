package app

import (
	"net/http"
	"slices"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
)

// Discovery detail handlers — status=public only.

func (app *App) showWork(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	work, err := app.getWork(r, "public")
	if err != nil {
		return err
	}
	return views.ShowWork(work).Render(r.Context(), w)
}

func (app *App) showPerson(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	person, err := app.getPerson(r)
	if err != nil {
		return err
	}
	return views.ShowPerson(person).Render(r.Context(), w)
}

func (app *App) showProject(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	project, err := app.getProject(r, "public")
	if err != nil {
		return err
	}
	return views.ShowProject(project).Render(r.Context(), w)
}

func (app *App) showOrganization(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	org, err := app.getOrganization(r)
	if err != nil {
		return err
	}
	return views.ShowOrganization(org).Render(r.Context(), w)
}

// Backoffice detail handlers — everything except deleted.

func (app *App) backofficeShowWork(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	work, err := app.getWork(r, "public", "private")
	if err != nil {
		return err
	}
	return views.BackofficeShowWork(work).Render(r.Context(), w)
}

func (app *App) backofficeShowPerson(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	person, err := app.getPerson(r)
	if err != nil {
		return err
	}
	return views.BackofficeShowPerson(person).Render(r.Context(), w)
}

func (app *App) backofficeShowProject(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	project, err := app.getProject(r, "public", "private")
	if err != nil {
		return err
	}
	return views.BackofficeShowProject(project).Render(r.Context(), w)
}

func (app *App) backofficeShowOrganization(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	org, err := app.getOrganization(r)
	if err != nil {
		return err
	}
	return views.BackofficeShowOrganization(org).Render(r.Context(), w)
}

// Entity fetchers with status checking.

func (app *App) getWork(r *http.Request, allowedStatuses ...string) (*bbl.Work, error) {
	id, err := bbl.ParseID(r.PathValue("id"))
	if err != nil {
		return nil, bbl.ErrNotFound
	}
	work, err := app.services.Repo.GetWork(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(allowedStatuses, work.Status) {
		return nil, bbl.ErrNotFound
	}
	return work, nil
}

func (app *App) getPerson(r *http.Request) (*bbl.Person, error) {
	id, err := bbl.ParseID(r.PathValue("id"))
	if err != nil {
		return nil, bbl.ErrNotFound
	}
	return app.services.Repo.GetPerson(r.Context(), id)
}

func (app *App) getProject(r *http.Request, allowedStatuses ...string) (*bbl.Project, error) {
	id, err := bbl.ParseID(r.PathValue("id"))
	if err != nil {
		return nil, bbl.ErrNotFound
	}
	project, err := app.services.Repo.GetProject(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if !slices.Contains(allowedStatuses, project.Status) {
		return nil, bbl.ErrNotFound
	}
	return project, nil
}

func (app *App) getOrganization(r *http.Request) (*bbl.Organization, error) {
	id, err := bbl.ParseID(r.PathValue("id"))
	if err != nil {
		return nil, bbl.ErrNotFound
	}
	return app.services.Repo.GetOrganization(r.Context(), id)
}
