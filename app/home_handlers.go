package app

import (
	"net/http"

	"github.com/ugent-library/bbl/app/views"
)

func (app *App) home(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	return views.Home().Render(r.Context(), w)
}

func (app *App) backofficeHome(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	return views.BackofficeHome().Render(r.Context(), w)
}
