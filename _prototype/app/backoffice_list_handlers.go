package app

import (
	"net/http"
	"time"

	"github.com/ugent-library/bbl/app/urls"
	"github.com/ugent-library/bbl/app/views"
	listviews "github.com/ugent-library/bbl/app/views/backoffice/lists"
	"github.com/ugent-library/bbl/can"
	"github.com/ugent-library/bbl/tasks"
	"github.com/ugent-library/catbird"
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
	_, err := app.repo.Catbird.RunTask(r.Context(), "export_works", tasks.ExportWorksInput{
		UserID: c.User.ID,
		ListID: id,
		Format: format,
	}, catbird.RunTaskOpts{})
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

func (app *App) backofficeAddListItems(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	r.ParseForm()

	lists, err := app.repo.GetUserLists(r.Context(), c.User.ID)
	if err != nil {
		return err
	}

	return listviews.AddItem(c.viewCtx(), lists, r.Form).Render(r.Context(), w)
}

// TODO check rights
func (app *App) backofficeCreateListItems(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	r.ParseForm()
	targetListID := r.PathValue("id")

	input := tasks.AddListItemsInput{
		UserID:       c.User.ID,
		TargetListID: targetListID,
	}

	if workIDs := r.Form["work_id"]; len(workIDs) > 0 {
		input.WorkIDs = workIDs
	} else if listID := r.FormValue("list_id"); listID != "" {
		input.ListID = listID
	} else {
		// TODO this duplicates backofficeExportWorks
		scope := r.FormValue("scope")
		if scope == "" {
			if can.Curate(c.User) {
				scope = "curator"
			} else {
				scope = "contributor"
			}

		}

		opts, err := app.bindSearchWorksOpts(r, c, scope)
		if err != nil {
			return err
		}

		// TODO
		opts.From = 0
		opts.Cursor = ""
		opts.Size = 100
		opts.Facets = nil

		input.SearchOpts = opts
	}

	// TODO do something with ref
	_, err := app.repo.Catbird.RunTask(r.Context(), tasks.AddListItemsName, input, catbird.RunTaskOpts{})
	if err != nil {
		return err
	}

	return nil
}
