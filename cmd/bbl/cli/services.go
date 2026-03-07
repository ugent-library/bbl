package cli

import (
	"context"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/ldap"
)

// Registry carries code-level configuration that cannot come from a config file.
// Pass a populated Registry to NewRootCmd in main.go.
// Use RegisterUserSource to register custom named user sources.
// Built-in source types (e.g. "ldap") are wired automatically from [user_sources] config.
type Registry struct {
	userSourceFactories map[string]func(*Config, toml.MetaData) (bbl.UserSource, error)
}

// RegisterUserSource registers a factory for a specific named user source.
// C must match the TOML structure under [user_sources.name.config].
// Named registrations take precedence over built-in type wiring.
func RegisterUserSource[C any](r *Registry, name string, fn func(C) (bbl.UserSource, error)) {
	if r.userSourceFactories == nil {
		r.userSourceFactories = make(map[string]func(*Config, toml.MetaData) (bbl.UserSource, error))
	}
	r.userSourceFactories[name] = func(cfg *Config, md toml.MetaData) (bbl.UserSource, error) {
		var c C
		if sc, ok := cfg.UserSources[name]; ok {
			if err := md.PrimitiveDecode(sc.Config, &c); err != nil {
				return nil, fmt.Errorf("user source %q: decode config: %w", name, err)
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
	cfg  Config
	reg Registry
	svc  *bbl.Services
}

// services lazily builds and caches *bbl.Services the first time it is called.
// Safe to call multiple times; always returns the same instance.
func (e *env) services(ctx context.Context) (*bbl.Services, error) {
	if e.svc != nil {
		return e.svc, nil
	}
	svc, err := newServices(ctx, &e.cfg, e.reg)
	if err != nil {
		return nil, err
	}
	e.svc = svc
	return svc, nil
}

func newServices(ctx context.Context, cfg *Config, reg Registry) (*bbl.Services, error) {
	if cfg.Conn == "" {
		return nil, fmt.Errorf("database connection string required (--conn or $BBL_CONN)")
	}

	repo, err := bbl.NewRepo(ctx, cfg.Conn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	n := len(reg.userSourceFactories) + len(cfg.UserSources)
	sources := make(map[string]bbl.UserSource, n)

	// Named factories: highest priority.
	for name, factory := range reg.userSourceFactories {
		src, err := factory(cfg, cfg.meta)
		if err != nil {
			repo.Close()
			return nil, fmt.Errorf("user source %q: %w", name, err)
		}
		sources[name] = src
	}

	// Built-in types: fallback for config-declared sources not registered by name.
	for name, sc := range cfg.UserSources {
		if _, ok := sources[name]; ok {
			continue
		}
		src, err := buildUserSource(name, sc, cfg.meta)
		if err != nil {
			repo.Close()
			return nil, err
		}
		sources[name] = src
	}

	return &bbl.Services{Repo: repo, Sources: sources}, nil
}

func buildUserSource(name string, sc UserSourceConfig, md toml.MetaData) (bbl.UserSource, error) {
	switch sc.Type {
	case "ldap":
		var cfg ldap.Config
		if err := md.PrimitiveDecode(sc.Config, &cfg); err != nil {
			return nil, fmt.Errorf("user source %q: decode config: %w", name, err)
		}
		return ldap.New(cfg), nil
	default:
		return nil, fmt.Errorf("user source %q: unknown type %q", name, sc.Type)
	}
}

