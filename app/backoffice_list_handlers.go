package app

import (
	"net/http"

	listviews "github.com/ugent-library/bbl/app/views/backoffice/lists"
)

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

func (app *App) backofficeAddListItem(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	workID := r.FormValue("work_id")

	lists, err := app.repo.GetUserLists(r.Context(), c.User.ID)
	if err != nil {
		return err
	}

	return listviews.AddItem(c.viewCtx(), lists, workID).Render(r.Context(), w)
}

func (app *App) backofficeCreateListItems(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	r.ParseForm()
	listID := r.PathValue("id")
	workIDs := r.Form["work_id"]

	if err := app.repo.AddListItems(r.Context(), listID, workIDs); err != nil {
		return err
	}

	return nil
}
