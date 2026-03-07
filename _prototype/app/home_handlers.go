package app

import (
	"net/http"

	discoveryviews "github.com/ugent-library/bbl/app/views/discovery"
)

func (app *App) home(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return discoveryviews.Home(c.viewCtx()).Render(r.Context(), w)
}
