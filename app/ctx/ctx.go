package ctx

import (
	"context"
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

type SessionCookie struct {
	UserID string `json:"u"`
}

type Config struct {
	Router     *mux.Router
	Hub        *catbird.Hub
	Assets     map[string]string
	Insecure   bool
	HashSecret []byte
	Secret     []byte
	UserFunc   func(context.Context, string) (*bbl.User, error)
}

type Ctx struct {
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

func New(config Config) *ctx.Ctx[*Ctx] {
	cookies := securecookie.New(config.HashSecret, config.Secret)
	cookies.SetSerializer(securecookie.JSONEncoder{})

	loc := i18n.Locales["en"] // TODO hardcoded for now

	crypter := crypt.New(config.Secret)

	return ctx.New(func(r *http.Request) (*Ctx, error) {
		c := &Ctx{
			Crypt:        crypter,
			URL:          r.URL,
			RouteName:    mux.CurrentRoute(r).GetName(),
			router:       config.Router,
			Hub:          config.Hub,
			secureCookie: cookies,
			assets:       config.Assets,
			insecure:     config.Insecure,
			Loc:          loc,
		}

		// get user from session if present
		session := SessionCookie{}
		err := c.GetCookie(r, sessionCookieName, &session)
		if err != nil && !errors.Is(err, http.ErrNoCookie) {
			return nil, err
		}
		if err == nil {
			user, err := config.UserFunc(r.Context(), session.UserID)
			if err != nil {
				return nil, err
			}
			c.User = user
			c.AddTopic("users")
			c.AddTopic("users." + user.ID)
		}

		return c, nil
	})
}

func (c *Ctx) ViewCtx() views.Ctx {
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

func (c *Ctx) AssetPath(asset string) string {
	a, ok := c.assets[asset]
	if !ok {
		panic(fmt.Errorf("asset '%s' not found in manifest", asset))
	}
	return a
}

func (c *Ctx) AddTopic(topic string) {
	if !slices.Contains(c.topics, topic) {
		c.topics = append(c.topics, topic)
	}
}

func (c *Ctx) SSEPath() string {
	token, err := c.EncryptValue(c.topics)
	if err != nil {
		panic(err)
	}
	return c.Route("sse", "token", token).String()
}

// TODO accept other types than string?
// TODO split off together with a url builder
func (c *Ctx) Route(name string, pairs ...string) *url.URL {
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

func (c *Ctx) SetUser(w http.ResponseWriter, user *bbl.User) error {
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

func (c *Ctx) ClearUser(w http.ResponseWriter) {
	c.User = nil
	c.ClearCookie(w, sessionCookieName)
}

func (c *Ctx) GetCookie(r *http.Request, name string, val any) error {
	cookie, err := r.Cookie(name)
	if err != nil {
		return fmt.Errorf("can't get cookie %s: %w", name, err)
	}
	if err := c.secureCookie.Decode(name, cookie.Value, val); err != nil {
		return fmt.Errorf("can't decode cookie %s: %w", name, err)
	}
	return nil
}

func (c *Ctx) SetCookie(w http.ResponseWriter, name string, val any, ttl time.Duration) error {
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

func (c *Ctx) ClearCookie(w http.ResponseWriter, name string) {
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
