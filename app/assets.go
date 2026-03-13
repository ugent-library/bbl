package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

//go:embed static
var staticFS embed.FS

// assets handles asset path resolution and file serving.
// In production: manifest is loaded once from the embedded FS.
// In dev mode: manifest is re-read from disk on every call to Path,
// and files are served from disk (so esbuild watch changes are instant).
type assets struct {
	prefix   string // path prefix (e.g. "/bbl" or "")
	dev      bool
	manifest map[string]string // production only
	dir      string           // dev only: path to static dir on disk
}

func loadAssets(prefix string, dev bool) (*assets, error) {
	a := &assets{prefix: prefix, dev: dev}

	if dev {
		// Find the static dir relative to the working directory.
		a.dir = filepath.Join("app", "static")
		return a, nil
	}

	// Production: load manifest once from embedded FS.
	data, err := staticFS.ReadFile("static/manifest.json")
	if err != nil {
		a.manifest = make(map[string]string)
		return a, nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse asset manifest: %w", err)
	}
	a.manifest = m
	return a, nil
}

// Path returns the URL path for a named asset (e.g. "app.css" → "/static/app-HASH.css").
// In dev mode, re-reads the manifest from disk each time.
func (a *assets) Path(name string) string {
	m := a.manifest
	if a.dev {
		m = a.readManifest()
	}
	p, ok := m[name]
	if !ok {
		// In dev mode with watch, the manifest may not exist yet.
		// Fall back to the unhashed name.
		if a.dev {
			return a.prefix + "/static/" + name
		}
		panic(fmt.Sprintf("asset %q not found in manifest", name))
	}
	return a.prefix + "/static/" + p
}

// fileServer returns an http.Handler that serves static files.
// Production: embedded FS with immutable cache headers.
// Dev: disk FS with no-cache headers (instant reload).
func (a *assets) fileServer() http.Handler {
	if a.dev {
		fileServer := http.FileServer(http.Dir(a.dir))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, r)
		})
	}

	sub, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}

func (a *assets) readManifest() map[string]string {
	data, err := os.ReadFile(filepath.Join(a.dir, "manifest.json"))
	if err != nil {
		return nil
	}
	m := make(map[string]string)
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
