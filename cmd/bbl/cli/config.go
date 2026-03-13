package cli

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"

	opensearch "github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/app"
	"github.com/ugent-library/bbl/arxiv"
	"github.com/ugent-library/bbl/ldap"
	"github.com/ugent-library/bbl/opensearchindex"
	"gopkg.in/yaml.v3"
)

type config struct {
	Conn       string           `yaml:"conn"`
	RootURL    string           `yaml:"root_url"`   // public root URL (e.g. "https://bbl.ugent.be")
	Dev        bool             `yaml:"dev"`         // dev mode: serve assets from disk, no caching
	OpenSearch openSearchConfig `yaml:"opensearch"` // OpenSearch connection config

	// Session secrets for cookie signing/encryption.
	HashSecret string `yaml:"hash_secret"` // HMAC key (hex-encoded or raw)
	Secret     string `yaml:"secret"`      // encryption key (hex-encoded or raw)
	Secure     bool   `yaml:"secure"`      // true = HTTPS-only cookies

	// Token encryption key (hex-encoded, 32 bytes / 64 hex chars) for AES-256-GCM.
	TokenSecret string `yaml:"token_secret"`

	// Path to the work profiles YAML file.
	ProfilePath string `yaml:"profiles"`

	// Populated from the config file:
	AuthProviders map[string]authProviderConfig `yaml:"auth"`
	UserSources   map[string]userSourceConfig   `yaml:"user_sources"`
	WorkSources   map[string]workSourceConfig   `yaml:"work_sources"`
}

type openSearchConfig struct {
	Addresses []string `yaml:"addresses"` // e.g. ["http://localhost:9200"]
}

type authProviderConfig struct {
	Type   string    `yaml:"type"`   // e.g. "oidc"
	Config yaml.Node `yaml:"config"` // decoded by RegisterAuthProvider
}

type userSourceConfig struct {
	Type         string    `yaml:"type"`          // informational, e.g. "ldap"
	AuthProvider string    `yaml:"auth_provider"` // optional auth provider name
	Config       yaml.Node `yaml:"config"`        // decoded by RegisterUserSource
}

type workSourceConfig struct {
	Type   string    `yaml:"type"`   // informational, e.g. "plato"
	Config yaml.Node `yaml:"config"` // decoded by RegisterWorkSource
}

// loadConfig reads the YAML config file at path, expands $VAR / ${VAR} references
// using the process environment, then decodes the result into cfg.
func loadConfig(path string, cfg *config) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(os.ExpandEnv(string(b))), cfg)
}

// Registry carries code-level configuration that cannot come from a config file.
// Pass a populated Registry to NewRootCmd in main.go.
// Use RegisterUserSource to register custom named user sources.
// Built-in source types (e.g. "ldap") are wired automatically from user_sources config.
type Registry struct {
	authProviderFactories map[string]func(*config) (app.AuthProvider, error)
	userSourceFactories   map[string]func(*config) (bbl.UserSource, error)
	workIterFactories     map[string]func(*config) (bbl.WorkSourceIter, error)
	workGetterFactories   map[string]func(*config) (bbl.WorkSourceGetter, error)
}

// RegisterUserSource registers a factory for a specific named user source.
// C must match the YAML structure under user_sources.<name>.config.
// Named registrations take precedence over built-in type wiring.
func RegisterUserSource[C any](r *Registry, name string, fn func(C) (bbl.UserSource, error)) {
	if r.userSourceFactories == nil {
		r.userSourceFactories = make(map[string]func(*config) (bbl.UserSource, error))
	}
	r.userSourceFactories[name] = func(cfg *config) (bbl.UserSource, error) {
		var c C
		if sc, ok := cfg.UserSources[name]; ok {
			if err := sc.Config.Decode(&c); err != nil {
				return nil, fmt.Errorf("user source %q: decode config: %w", name, err)
			}
		}
		return fn(c)
	}
}

// RegisterWorkSourceIter registers a factory for a named work iteration source.
// C must match the YAML structure under work_sources.<name>.config.
func RegisterWorkSourceIter[C any](r *Registry, name string, fn func(C) (bbl.WorkSourceIter, error)) {
	if r.workIterFactories == nil {
		r.workIterFactories = make(map[string]func(*config) (bbl.WorkSourceIter, error))
	}
	r.workIterFactories[name] = func(cfg *config) (bbl.WorkSourceIter, error) {
		var c C
		if sc, ok := cfg.WorkSources[name]; ok {
			if err := sc.Config.Decode(&c); err != nil {
				return nil, fmt.Errorf("work source %q: decode config: %w", name, err)
			}
		}
		return fn(c)
	}
}

// RegisterWorkSourceGetter registers a factory for a named work get source.
// C must match the YAML structure under work_sources.<name>.config.
func RegisterWorkSourceGetter[C any](r *Registry, name string, fn func(C) (bbl.WorkSourceGetter, error)) {
	if r.workGetterFactories == nil {
		r.workGetterFactories = make(map[string]func(*config) (bbl.WorkSourceGetter, error))
	}
	r.workGetterFactories[name] = func(cfg *config) (bbl.WorkSourceGetter, error) {
		var c C
		if sc, ok := cfg.WorkSources[name]; ok {
			if err := sc.Config.Decode(&c); err != nil {
				return nil, fmt.Errorf("work source %q: decode config: %w", name, err)
			}
		}
		return fn(c)
	}
}

// RegisterAuthProvider registers a factory for a named auth provider.
// C must match the YAML structure under auth.<name>.config.
func RegisterAuthProvider[C any](r *Registry, name string, fn func(C) (app.AuthProvider, error)) {
	if r.authProviderFactories == nil {
		r.authProviderFactories = make(map[string]func(*config) (app.AuthProvider, error))
	}
	r.authProviderFactories[name] = func(cfg *config) (app.AuthProvider, error) {
		var c C
		if ac, ok := cfg.AuthProviders[name]; ok {
			if err := ac.Config.Decode(&c); err != nil {
				return nil, fmt.Errorf("auth provider %q: decode config: %w", name, err)
			}
		}
		return fn(c)
	}
}

// env is the shared state for all CLI commands. It holds configuration and
// options set before execution, and lazily constructs bbl.Services on first use.
// Commands that need the database or sources call env.services(ctx); commands
// that do not (e.g. migrate) call nothing and pay no connection cost.
type env struct {
	cfg *config
	reg *Registry
	svc *bbl.Services
}

// services lazily builds and caches *bbl.Services the first time it is called.
// Safe to call multiple times; always returns the same instance.
func (e *env) services(ctx context.Context) (*bbl.Services, error) {
	if e.svc != nil {
		return e.svc, nil
	}
	svc, err := e.newServices(ctx)
	if err != nil {
		return nil, err
	}
	e.svc = svc
	return svc, nil
}

func (e *env) newServices(ctx context.Context) (*bbl.Services, error) {
	cfg, reg := e.cfg, e.reg

	if cfg.Conn == "" {
		return nil, fmt.Errorf("database connection string required (--conn or $BBL_CONN)")
	}

	var tokenKey []byte
	if cfg.TokenSecret != "" {
		var err error
		tokenKey, err = hex.DecodeString(cfg.TokenSecret)
		if err != nil {
			return nil, fmt.Errorf("token_secret: invalid hex: %w", err)
		}
		if len(tokenKey) != 32 {
			return nil, fmt.Errorf("token_secret: must be 32 bytes (64 hex chars), got %d bytes", len(tokenKey))
		}
	}

	repo, err := bbl.NewRepo(ctx, cfg.Conn, tokenKey)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// --- Work profiles (optional) ---
	if cfg.ProfilePath != "" {
		profiles, err := bbl.LoadWorkProfiles(cfg.ProfilePath)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("load profiles: %w", err)
		}
		repo.WorkProfiles = profiles
	}

	// --- User sources ---
	userSources := make(map[string]bbl.UserSource)

	for name, factory := range reg.userSourceFactories {
		src, err := factory(cfg)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("user source %q: %w", name, err)
		}
		userSources[name] = src
	}

	for name, sc := range cfg.UserSources {
		if _, ok := userSources[name]; ok {
			continue
		}
		var src bbl.UserSource
		switch sc.Type {
		case "ldap":
			var c ldap.Config
			if err := sc.Config.Decode(&c); err != nil {
				repo.Close()
				return nil, fmt.Errorf("user source %q: decode config: %w", name, err)
			}
			src = ldap.New(c)
		default:
			repo.Close()
			return nil, fmt.Errorf("user source %q: unknown type %q", name, sc.Type)
		}
		userSources[name] = src
	}

	// --- Work iter sources ---
	workIterSources := make(map[string]bbl.WorkSourceIter)

	for name, factory := range reg.workIterFactories {
		src, err := factory(cfg)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("work source %q: %w", name, err)
		}
		workIterSources[name] = src
	}

	for name, sc := range cfg.WorkSources {
		if _, ok := workIterSources[name]; ok {
			continue
		}
		repo.Close()
		return nil, fmt.Errorf("work source %q: unknown type %q", name, sc.Type)
	}

	// --- Work get sources (built-in) ---
	workGetSources := make(map[string]bbl.WorkSourceGetter)

	for name, factory := range reg.workGetterFactories {
		src, err := factory(cfg)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("work source %q: %w", name, err)
		}
		workGetSources[name] = src
	}

	workGetSources["arxiv"] = arxiv.NewWorkSource()

	// Seed built-in sources (curator, self_deposit) with default priorities.
	if err := repo.SeedBuiltinSources(ctx); err != nil {
		repo.Close()
		return nil, err
	}

	// Register all configured sources in bbl_sources so FK constraints are satisfied.
	for name := range cfg.UserSources {
		if err := repo.UpsertSource(ctx, name); err != nil {
			repo.Close()
			return nil, err
		}
	}
	for name := range workIterSources {
		if err := repo.UpsertSource(ctx, name); err != nil {
			repo.Close()
			return nil, err
		}
	}
	for name := range workGetSources {
		if err := repo.UpsertSource(ctx, name); err != nil {
			repo.Close()
			return nil, err
		}
	}

	// --- OpenSearch index (optional) ---
	var index bbl.Index
	if len(cfg.OpenSearch.Addresses) > 0 {
		osClient, err := opensearchapi.NewClient(opensearchapi.Config{
			Client: opensearch.Config{
				Addresses: cfg.OpenSearch.Addresses,
			},
		})
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("opensearch client: %w", err)
		}
		idx, err := opensearchindex.New(ctx, opensearchindex.Config{
			Client: osClient,
			OnFail: func(ctx context.Context, id string, err error) {
				slog.Error("opensearch", "id", id, "err", err)
			},
		})
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("opensearch index: %w", err)
		}
		index = idx
	}

	return &bbl.Services{
		Repo:            repo,
		Index:           index,
		UserSources:     userSources,
		WorkIterSources: workIterSources,
		WorkGetSources:  workGetSources,
	}, nil
}
