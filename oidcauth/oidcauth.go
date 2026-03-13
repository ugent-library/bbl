// Package oidcauth implements app.AuthProvider using OpenID Connect.
// It uses coreos/go-oidc for discovery and token verification, and
// golang.org/x/oauth2 for the authorization code flow.
package oidcauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/securecookie"
	"github.com/ugent-library/bbl/app"
	"golang.org/x/oauth2"
)

const cookieMaxAge = time.Hour

// Config holds the OIDC provider configuration.
type Config struct {
	IssuerURL    string `yaml:"issuer_url"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`

	// Claim is the OIDC token claim to extract (e.g. "sub", "preferred_username", or a custom claim).
	Claim string `yaml:"claim"`
	// Match is how the extracted claim maps to a user: "username" for bbl_users.username,
	// or an identifier scheme name (e.g. "ugent_id") for bbl_user_identifiers.
	Match string `yaml:"match"`
}

// Provider implements app.AuthProvider using OIDC.
type Provider struct {
	oauth2Config *oauth2.Config
	verifier     *gooidc.IDTokenVerifier
	cookies      *securecookie.SecureCookie
	secure       bool
	stateCookie  string
	nonceCookie  string
	claim        string
	match        string
}

// New creates an OIDC auth provider. It performs OIDC discovery synchronously.
// cookieHashSecret and cookieSecret are used to sign/encrypt the short-lived
// state and nonce cookies for the auth flow. secure controls the Secure flag
// on those cookies (should be true in production).
func New(ctx context.Context, c Config, cookieHashSecret, cookieSecret []byte, secure bool) (*Provider, error) {
	if c.IssuerURL == "" {
		return nil, errors.New("oidcauth: issuer_url required")
	}
	if c.ClientID == "" {
		return nil, errors.New("oidcauth: client_id required")
	}
	if c.RedirectURL == "" {
		return nil, errors.New("oidcauth: redirect_url required")
	}
	if c.Claim == "" {
		c.Claim = "sub"
	}
	if c.Match == "" {
		return nil, errors.New("oidcauth: match required")
	}

	provider, err := gooidc.NewProvider(ctx, c.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: discovery: %w", err)
	}

	oauth2Config := &oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		RedirectURL:  c.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID},
	}

	verifier := provider.Verifier(&gooidc.Config{ClientID: c.ClientID})

	return &Provider{
		oauth2Config: oauth2Config,
		verifier:     verifier,
		cookies:      securecookie.New(cookieHashSecret, cookieSecret),
		secure:       secure,
		stateCookie:  "bbl.oidc.state",
		nonceCookie:  "bbl.oidc.nonce",
		claim:        c.Claim,
		match:        c.Match,
	}, nil
}

// BeginAuth redirects the user to the OIDC provider's authorization endpoint.
func (p *Provider) BeginAuth(w http.ResponseWriter, r *http.Request) error {
	state, err := randomString(32)
	if err != nil {
		return fmt.Errorf("oidcauth: generate state: %w", err)
	}
	nonce, err := randomString(32)
	if err != nil {
		return fmt.Errorf("oidcauth: generate nonce: %w", err)
	}

	if err := p.setCookie(w, p.stateCookie, state); err != nil {
		return err
	}
	if err := p.setCookie(w, p.nonceCookie, nonce); err != nil {
		return err
	}

	url := p.oauth2Config.AuthCodeURL(state, gooidc.Nonce(nonce))
	http.Redirect(w, r, url, http.StatusFound)
	return nil
}

// CompleteAuth handles the OIDC callback, verifies the ID token, and extracts
// the configured claim into an AuthResult.
func (p *Provider) CompleteAuth(w http.ResponseWriter, r *http.Request) (*app.AuthResult, error) {
	state, err := p.getCookie(r, p.stateCookie)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: state cookie: %w", err)
	}
	nonce, err := p.getCookie(r, p.nonceCookie)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: nonce cookie: %w", err)
	}

	// Clear auth cookies regardless of outcome.
	p.clearCookie(w, p.stateCookie)
	p.clearCookie(w, p.nonceCookie)

	if r.URL.Query().Get("state") != state {
		return nil, errors.New("oidcauth: invalid state")
	}

	oauthToken, err := p.oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		return nil, fmt.Errorf("oidcauth: token exchange: %w", err)
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("oidcauth: id_token missing from token response")
	}

	idToken, err := p.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidcauth: verify id_token: %w", err)
	}

	if idToken.Nonce != nonce {
		return nil, errors.New("oidcauth: invalid nonce")
	}

	// Extract the configured claim.
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("oidcauth: decode claims: %w", err)
	}

	val, ok := claims[p.claim]
	if !ok {
		return nil, fmt.Errorf("oidcauth: claim %q not present in id_token", p.claim)
	}

	strVal, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("oidcauth: claim %q is not a string", p.claim)
	}
	if strVal == "" {
		return nil, fmt.Errorf("oidcauth: claim %q is empty", p.claim)
	}

	return &app.AuthResult{
		Match: p.match,
		Value: strVal,
	}, nil
}

func (p *Provider) setCookie(w http.ResponseWriter, name, val string) error {
	encoded, err := p.cookies.Encode(name, val)
	if err != nil {
		return fmt.Errorf("oidcauth: encode cookie %s: %w", name, err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    encoded,
		Path:     "/",
		MaxAge:   int(cookieMaxAge / time.Second),
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (p *Provider) getCookie(r *http.Request, name string) (string, error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}
	var val string
	if err := p.cookies.Decode(name, cookie.Value, &val); err != nil {
		return "", err
	}
	return val, nil
}

func (p *Provider) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   p.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
