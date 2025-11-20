package app

import (
	"encoding/json"
	"net/http"

	"github.com/tidwall/gjson"
	"github.com/ugent-library/bbl/app/urls"
	"github.com/ugent-library/bbl/httperr"
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

	c.User = user
	if err = c.SaveSession(w); err != nil {
		return err
	}

	http.Redirect(w, r, urls.BackofficeHome(), http.StatusFound)

	return nil
}

func (app *App) logout(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	if c.ViewAsUser != nil {
		c.User = c.ViewAsUser
		c.ViewAsUser = nil
		if err := c.SaveSession(w); err != nil {
			return err
		}
	} else {
		c.User = nil
		c.ClearCookie(w, sessionCookieName)
	}

	http.Redirect(w, r, urls.BackofficeHome(), http.StatusFound)

	return nil
}

// TODO check rights
func (app *App) viewAs(w http.ResponseWriter, r *http.Request, c *appCtx) error {
	if c.ViewAsUser != nil {
		return httperr.BadRequest
	}

	user, err := app.repo.GetUser(r.Context(), r.FormValue("user_id"))
	if err != nil {
		return err
	}

	if user.ID == c.User.ID {
		return httperr.BadRequest
	}

	c.ViewAsUser = c.User
	c.User = user

	if err = c.SaveSession(w); err != nil {
		return err
	}

	http.Redirect(w, r, urls.BackofficeHome(), http.StatusFound)

	return nil
}
