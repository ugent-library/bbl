package app

import (
	"net/http"
	"time"

	"github.com/ugent-library/bbl/app/urls"
	"github.com/ugent-library/bbl/app/views"
	listviews "github.com/ugent-library/bbl/app/views/backoffice/lists"
	"github.com/ugent-library/bbl/workflows"
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

func (app *App) backofficeList(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	id := r.PathValue("id")

	list, err := app.repo.GetList(r.Context(), id)
	if err != nil {
		return err
	}

	listItems, err := app.repo.GetListItems(r.Context(), id)
	if err != nil {
		return err
	}

	return listviews.Show(c.viewCtx(), list, listItems).Render(r.Context(), w)
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
func (app *App) backofficeExportList(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	id := r.PathValue("id")
	format := r.PathValue("format")

	// TODO do something with ref
	_, err := app.exportWorksTask.RunNoWait(r.Context(), workflows.ExportWorksInput{
		UserID: c.User.ID,
		ListID: id,
		Format: format,
	})
	if err != nil {
		return err
	}

	return views.AddFlash(views.FlashArgs{
		Type:         views.FlashInfo,
		Title:        "Export started",
		Text:         "You will be notified when your export is ready.",
		DismissAfter: 5 * time.Second,
	}).Render(r.Context(), w)
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
