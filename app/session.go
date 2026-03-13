package app

import (
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
)

const (
	sessionCookieName = "bbl.session"
	sessionMaxAge     = 30 * 24 * int(time.Hour/time.Second) // 30 days
)

type sessionData struct {
	UserID string `json:"u,omitempty"`
}

type session struct {
	cookies *securecookie.SecureCookie
	secure  bool
}

func newSession(hashSecret, secret []byte, secure bool) *session {
	sc := securecookie.New(hashSecret, secret)
	sc.SetSerializer(securecookie.JSONEncoder{})
	sc.MaxAge(sessionMaxAge) // reject encoded values older than this
	return &session{cookies: sc, secure: secure}
}

func (s *session) load(r *http.Request) (*sessionData, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return &sessionData{}, nil
	}
	if err != nil {
		return nil, err
	}
	var data sessionData
	if err := s.cookies.Decode(sessionCookieName, cookie.Value, &data); err != nil {
		// Corrupted or expired cookie — treat as no session.
		return &sessionData{}, nil
	}
	return &data, nil
}

func (s *session) save(w http.ResponseWriter, data *sessionData) error {
	encoded, err := s.cookies.Encode(sessionCookieName, data)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (s *session) clear(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}
