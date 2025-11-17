package app

import (
	"net/http"

	"github.com/ugent-library/bbl/app/urls"
	listviews "github.com/ugent-library/bbl/app/views/backoffice/lists"
	"github.com/ugent-library/htmx"
)

func (app *App) backofficeLists(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	lists, err := app.repo.GetUserLists(r.Context(), c.User.ID)
	if err != nil {
		return err
	}
	return listviews.Overview(c.viewCtx(), lists).Render(r.Context(), w)
}

func (app *App) backofficeNewList(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return listviews.New(c.viewCtx()).Render(r.Context(), w)
}

func (app *App) backofficeCreateList(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	name := r.FormValue("name")

	if _, err := app.repo.CreateList(r.Context(), c.User.ID, name); err != nil {
		return err
	}

	return nil
}

// TODO check rights
func (app *App) backofficeDeleteList(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	id := r.PathValue("id")

	if err := app.repo.DeleteList(r.Context(), id); err != nil {
		return err
	}

	htmx.Redirect(w, urls.BackofficeLists())

	return nil
}

// TODO check rights
func (app *App) backofficeAddListItem(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	workID := r.FormValue("work_id")

	lists, err := app.repo.GetUserLists(r.Context(), c.User.ID)
	if err != nil {
		return err
	}

	return listviews.AddItem(c.viewCtx(), lists, workID).Render(r.Context(), w)
}

// TODO check rights
func (app *App) backofficeCreateListItems(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	r.ParseForm()
	listID := r.PathValue("id")
	workIDs := r.Form["work_id"]

	if err := app.repo.AddListItems(r.Context(), listID, workIDs); err != nil {
		return err
	}

	return nil
}
