package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config Config

func RunWithContext(ctx context.Context) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("bbl")
	v.SetDefault("env", "production")
	v.SetDefault("port", 3000)
	v.BindEnv("env")
	v.BindEnv("host")
	v.BindEnv("port")
	v.BindEnv("base_url")
	v.BindEnv("pg_conn")
	v.BindEnv("opensearch.url")
	v.BindEnv("opensearch.username")
	v.BindEnv("opensearch.password")
	v.BindEnv("oidc.issuer_url")
	v.BindEnv("oidc.client_id")
	v.BindEnv("oidc.client_secret")
	v.BindEnv("cookie_secret")
	v.BindEnv("cookie_hash_secret")

	if err := v.Unmarshal(&config); err != nil {
		return err
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		return err
	}

	return nil
}

func Run() {
	cobra.CheckErr(RunWithContext(context.Background()))
}
