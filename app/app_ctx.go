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
	"github.com/leonelquinteros/gotext"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app/views"
	"github.com/ugent-library/bbl/catbird"
	"github.com/ugent-library/bbl/ctx"
	"github.com/ugent-library/bbl/i18n"
	"github.com/ugent-library/crypt"
)

const (
	sessionCookieName = "bbl.session"
)

func RequireUser(w http.ResponseWriter, r *http.Request, c *AppCtx) (*http.Request, error) {
	if c.User == nil {
		http.Redirect(w, r, c.Route("login").String(), http.StatusFound)
		return nil, nil
	}
	return r, nil
}

type SessionCookie struct {
	UserID string `json:"u"`
}

type AppCtx struct {
	router       *mux.Router
	Hub          *catbird.Hub
	topics       []string
	secureCookie *securecookie.SecureCookie
	assets       map[string]string
	insecure     bool
	*crypt.Crypt
	Loc       *gotext.Locale
	URL       *url.URL
	RouteName string
	User      *bbl.User
}

func BindAppCtx(config *Config, router *mux.Router, assets map[string]string) ctx.Binder[*AppCtx] {
	cookies := securecookie.New(config.HashSecret, config.Secret)
	cookies.SetSerializer(securecookie.JSONEncoder{})

	insecure := config.Env == "development"

	loc := i18n.Locales["en"] // TODO hardcoded for now

	userFunc := config.Repo.GetUser

	crypter := crypt.New(config.Secret)

	return func(r *http.Request) (*AppCtx, error) {
		c := &AppCtx{
			Crypt:        crypter,
			URL:          r.URL,
			RouteName:    mux.CurrentRoute(r).GetName(),
			router:       router,
			Hub:          config.Hub,
			secureCookie: cookies,
			assets:       assets,
			insecure:     insecure,
			Loc:          loc,
		}

		// get user from session if present
		session := SessionCookie{}
		err := c.GetCookie(r, sessionCookieName, &session)
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			return nil, err
		}
		if err == nil {
			user, err := userFunc(r.Context(), session.UserID)
			if err != nil {
				return nil, err
			}
			c.User = user
			c.AddTopic("users")
			c.AddTopic("users." + user.ID)
		}

		return c, nil
	}
}

func (c *AppCtx) ViewCtx() views.Ctx {
	return views.Ctx{
		URL:       c.URL,
		RouteName: c.RouteName,
		Route:     c.Route,
		AssetPath: c.AssetPath,
		SSEPath:   c.SSEPath,
		Loc:       c.Loc,
		User:      c.User,
	}
}

func (c *AppCtx) AssetPath(asset string) string {
	a, ok := c.assets[asset]
	if !ok {
		panic(fmt.Errorf("asset '%s' not found in manifest", asset))
	}
	return a
}

func (c *AppCtx) AddTopic(topic string) {
	if !slices.Contains(c.topics, topic) {
		c.topics = append(c.topics, topic)
	}
}

func (c *AppCtx) SSEPath() string {
	token, err := c.EncryptValue(c.topics)
	if err != nil {
		panic(err)
	}
	return c.Route("sse", "token", token).String()
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

func (c *AppCtx) SetUser(w http.ResponseWriter, user *bbl.User) error {
	val := &SessionCookie{
		UserID: user.ID,
	}
	err := c.SetCookie(w, sessionCookieName, val, 30*24*time.Hour)
	if err != nil {
		return err
	}

	c.User = user

	return nil
}

func (c *AppCtx) ClearUser(w http.ResponseWriter) {
	c.User = nil
	c.ClearCookie(w, sessionCookieName)
}

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
