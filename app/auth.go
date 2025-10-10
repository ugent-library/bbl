package app

import (
	"encoding/json"
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl/app/urls"
)

func (app *App) login(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	return app.authProvider.BeginAuth(w, r)
}

func (app *App) authCallback(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	var claims json.RawMessage

	if err := app.authProvider.CompleteAuth(w, r, &claims); err != nil {
		return err
	}

	username := gjson.GetBytes(claims, "sub").String() // TODO make username claim configurable

	user, err := app.repo.GetUser(r.Context(), "username:"+username)
	if err != nil {
		return err
	}

	if err = c.SetUser(w, user); err != nil {
		return err
	}

	http.Redirect(w, r, urls.BackofficeHome(), http.StatusFound)

	return nil
}

func (app *App) logout(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	c.ClearUser(w)
	http.Redirect(w, r, urls.BackofficeHome(), http.StatusFound)
	return nil
}
