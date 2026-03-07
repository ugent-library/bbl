package cli

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds all runtime configuration for the CLI.
// Simple values are populated from flags and environment variables in NewRootCmd.
// Adapter-specific config (sources, OIDC, …) is loaded from the TOML file
// pointed to by ConfigFile / $BBL_CONFIG.
// Commands receive a *Config and do not read flags or env directly.
type Config struct {
	Conn       string
	ConfigFile string

	// Populated from the config file:
	UserSources map[string]UserSourceConfig `toml:"user_sources"`

	// meta holds TOML decode metadata, used by RegisterUserSource to decode
	// [user_sources.name.config] sections into caller-supplied typed structs.
	meta toml.MetaData
}

// UserSourceConfig describes one named user source. Type is informational (e.g.
// for tooling or documentation); all wiring is done via RegisterUserSource.
// Connection parameters go under [user_sources.name.config] and are decoded into
// the caller's typed struct by RegisterUserSource via toml.PrimitiveDecode.
type UserSourceConfig struct {
	Type         string         `toml:"type"`          // informational, e.g. "ldap"
	AuthProvider string         `toml:"auth_provider"` // optional auth provider name
	Config       toml.Primitive `toml:"config"`        // decoded by RegisterUserSource
}

func defaultConfig() Config {
	return Config{
		Conn:       os.Getenv("BBL_CONN"),
		ConfigFile: os.Getenv("BBL_CONFIG"),
	}
}

// loadFile decodes the TOML config file into cfg, merging with any values
// already set from flags or environment variables. The TOML metadata is
// retained so that RegisterUserSource can later decode [user_sources.name.config]
// sections via toml.PrimitiveDecode.
func (cfg *Config) loadFile() error {
	md, err := toml.DecodeFile(cfg.ConfigFile, cfg)
	cfg.meta = md
	return err
}
