package app

import (
	"net/http"

	backofficeviews "github.com/ugent-library/bbl/app/views/backoffice"
)

func (app *App) backofficeHome(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return backofficeviews.Home(c.viewCtx()).Render(r.Context(), w)
}
