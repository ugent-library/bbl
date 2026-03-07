package app

import (
	"net/http"

	projectviews "github.com/ugent-library/bbl/app/views/backoffice/projects"
)

func (app *App) backofficeProjects(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	opts, err := bindSearchOpts(r, nil)
	if err != nil {
		return err
	}

	hits, err := app.index.Projects().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return projectviews.Search(c.viewCtx(), hits).Render(r.Context(), w)
}
