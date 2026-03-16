package app

import (
	"net/http"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
)

// Ctx holds per-request state. Just data — no methods, no infrastructure.
type Ctx struct {
	User *bbl.User
	ViewCtx views.Ctx
}

// newCtx builds a Ctx, loading the user from session if present.
// Used for public routes where auth is optional.
func (app *App) newCtx(r *http.Request) (*Ctx, error) {
	lang := app.locale.match(r.Header.Get("Accept-Language"), localeCookieValue(r), r.URL.Query().Get("lang"))
	c := &Ctx{
		ViewCtx: views.Ctx{
			AssetPath: app.assets.Path,
			Loc:       app.locale.translateFunc(lang),
			Lang:      lang,
			Langs:     app.locale.langs,
			Path:      r.URL.Path,
		},
	}
	if app.session == nil {
		return c, nil
	}
	sess, err := app.session.load(r)
	if err != nil {
		return nil, err
	}
	if sess.UserID != "" {
		id, err := bbl.ParseID(sess.UserID)
		if err != nil {
			// Bad ID in cookie — treat as no session.
			return c, nil
		}
		user, err := app.services.Repo.GetUser(r.Context(), id)
		if err != nil {
			if err == bbl.ErrNotFound {
				// User deleted — treat as no session.
				return c, nil
			}
			return nil, err
		}
		c.User = user
	}
	return c, nil
}

// newAuthCtx builds a Ctx and requires a logged-in user.
// Returns errNotAuthenticated if no user — the error handler maps this
// to a login redirect.
func (app *App) newAuthCtx(r *http.Request) (*Ctx, error) {
	c, err := app.newCtx(r)
	if err != nil {
		return nil, err
	}
	if c.User == nil {
		return nil, errNotAuthenticated
	}
	return c, nil
}
