package app

import (
	"net/http"

	backofficeviews "github.com/ugent-library/bbl/app/views/backoffice"
	"github.com/ugent-library/bbl/catbird"
)

func (app *App) backofficeHome(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return backofficeviews.Home(c.viewCtx()).Render(r.Context(), w)
}

func (app *App) backofficeSSE(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var topics []string
	if err := c.crypt.DecryptValue(r.URL.Query().Get("token"), &topics); err != nil {
		return err
	}
	return c.Hub.ConnectSSE(w, r, catbird.ConnectOpts{
		Topics: topics,
	})
}
