package app

import (
	"net/http"

	organizationviews "github.com/ugent-library/bbl/app/views/backoffice/organizations"
)

func (app *App) backofficeOrganizations(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	opts, err := bindSearchOpts(r, nil)
	if err != nil {
		return err
	}

	hits, err := app.index.Organizations().Search(r.Context(), opts)
	if err != nil {
		return err
	}

	return organizationviews.Search(c.viewCtx(), hits).Render(r.Context(), w)
}
