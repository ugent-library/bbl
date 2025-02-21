package app

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/ctx"
)

const (
	sessionCookieName = "bbl.session"
)

type SessionCookie struct {
	UserID string `json:"u"`
}

type AppCtx struct {
	router       *mux.Router
	secureCookie *securecookie.SecureCookie
	assets       map[string]string
	insecure     bool
	// user         *biblio.User
}

func BindAppCtx(router *mux.Router, cookies *securecookie.SecureCookie, assets map[string]string, insecure bool) ctx.Binder[*AppCtx] {
	// func BindAppCtx(router *mux.Router, cookies *securecookie.SecureCookie, assets map[string]string, insecure bool, usersRepo biblio.Users) ctx.Binder[*AppCtx] {
	return func(r *http.Request) (*AppCtx, error) {
		c := &AppCtx{
			router:       router,
			secureCookie: cookies,
			assets:       assets,
			insecure:     insecure,
		}

		// get user from session if present
		session := SessionCookie{}
		err := c.GetCookie(r, sessionCookieName, &session)
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			return nil, err
		}
		// if err == nil {
		// 	user, err := usersRepo.Get(r.Context(), session.UserID)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	c.user = user
		// }

		return c, nil
	}
}

func (c *AppCtx) ViewCtx() views.Ctx {
	return views.Ctx{
		Route:     c.Route,
		AssetPath: c.AssetPath,
		// User:      c.user,
	}
}

func (c *AppCtx) AssetPath(asset string) string {
	a, ok := c.assets[asset]
	if !ok {
		panic(fmt.Errorf("asset '%s' not found in manifest", asset))
	}
	return a
}

// TODO accept other types than string?
// TODO split off together with a url builder
func (c *AppCtx) Route(name string, pairs ...string) *url.URL {
	route := c.router.Get(name)
	if route == nil {
		panic(errors.New("unknown route " + name))
	}

	varsNames, err := route.GetVarNames()
	if err != nil {
		panic(err)
	}

	var vars []string
	var queryParams []string

	for i := 0; i+1 < len(pairs); i += 2 {
		if slices.Contains(varsNames, pairs[i]) {
			vars = append(vars, pairs[i], pairs[i+1])
		} else {
			queryParams = append(queryParams, pairs[i], pairs[i+1])
		}
	}

	u, err := route.URL(vars...)
	if err != nil {
		panic(err)
	}

	if len(queryParams) > 0 {
		q := u.Query()
		for i := 0; i < len(queryParams); i += 2 {
			q.Add(queryParams[i], queryParams[i+1])
		}
		u.RawQuery = q.Encode()
	}

	return u
}

// func (c *AppCtx) User() *biblio.User {
// 	return c.user
// }

// func (c *AppCtx) SetUser(w http.ResponseWriter, user *biblio.User) error {
// 	val := &SessionCookie{
// 		UserID: user.ID,
// 	}
// 	err := c.SetCookie(w, sessionCookieName, val, 30*24*time.Hour)
// 	if err != nil {
// 		return err
// 	}

// 	c.user = user

// 	return nil
// }

// func (c *AppCtx) ClearUser(w http.ResponseWriter) {
// 	c.user = nil
// 	c.ClearCookie(w, sessionCookieName)
// }

func (c *AppCtx) GetCookie(r *http.Request, name string, val any) error {
	cookie, err := r.Cookie(name)
	if err != nil {
		return fmt.Errorf("can't get cookie %s: %w", name, err)
	}
	if err := c.secureCookie.Decode(name, cookie.Value, val); err != nil {
		return fmt.Errorf("can't decode cookie %s: %w", name, err)
	}
	return nil
}

func (c *AppCtx) SetCookie(w http.ResponseWriter, name string, val any, ttl time.Duration) error {
	v, err := c.secureCookie.Encode(name, val)
	if err != nil {
		return fmt.Errorf("can't encode cookie %s: %w", name, err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    v,
		Path:     "/",
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   !c.insecure,
		SameSite: http.SameSiteStrictMode,
	})

	return nil
}

func (c *AppCtx) ClearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !c.insecure,
		SameSite: http.SameSiteStrictMode,
	})
}
