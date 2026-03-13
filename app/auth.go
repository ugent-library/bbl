package app

import (
	"net/http"

	"github.com/ugent-library/bbl"
)

// AuthProvider handles an external auth flow (e.g. OIDC).
// Each provider has a name (e.g. "ugent_oidc") used in routes and user records.
type AuthProvider interface {
	BeginAuth(http.ResponseWriter, *http.Request) error
	CompleteAuth(http.ResponseWriter, *http.Request) (*AuthResult, error)
}

// AuthResult is the outcome of a successful external auth flow.
// Match is either "username" (matched against bbl_users.username)
// or an identifier scheme name (matched against bbl_user_identifiers).
type AuthResult struct {
	Match string // "username" or identifier scheme (e.g. "ugent_id")
	Value string // the claim value (e.g. "abc123")
}

func (app *App) login(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	if len(app.auth) == 0 {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}
	// TODO render login page with provider choices
	// For now, if there's exactly one provider, begin auth directly.
	for _, provider := range app.auth {
		return provider.BeginAuth(w, r)
	}
	return nil
}

func (app *App) loginProvider(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	name := r.PathValue("provider")
	provider, ok := app.auth[name]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}
	return provider.BeginAuth(w, r)
}

func (app *App) authCallback(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	name := r.PathValue("provider")
	provider, ok := app.auth[name]
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return nil
	}

	result, err := provider.CompleteAuth(w, r)
	if err != nil {
		return err
	}

	var user *bbl.User
	if result.Match == "username" {
		user, err = app.services.Repo.GetUserByUsername(r.Context(), result.Value)
	} else {
		user, err = app.services.Repo.GetUserByIdentifier(r.Context(), result.Match, result.Value)
	}
	if err != nil {
		if err == bbl.ErrNotFound {
			http.Error(w, "Unknown user", http.StatusForbidden)
			return nil
		}
		return err
	}

	if err := app.session.save(w, &sessionData{UserID: user.ID.String()}); err != nil {
		return err
	}

	http.Redirect(w, r, "/backoffice", http.StatusFound)
	return nil
}

func (app *App) logout(w http.ResponseWriter, r *http.Request, c *Ctx) error {
	app.session.clear(w)
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
