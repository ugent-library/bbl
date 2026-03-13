package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	a, err := app.New(app.Config{
		Services: &bbl.Services{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(newTestApp(t).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHome(t *testing.T) {
	srv := httptest.NewServer(newTestApp(t).Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBackofficeRedirectsWhenUnauthenticated(t *testing.T) {
	srv := httptest.NewServer(newTestApp(t).Handler())
	defer srv.Close()

	client := &http.Client{CheckRedirect: func(r *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	resp, err := client.Get(srv.URL + "/backoffice")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if loc != "/backoffice/login" {
		t.Fatalf("expected redirect to /backoffice/login, got %s", loc)
	}
}
