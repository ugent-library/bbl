package cli

import (
	"context"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/csl"
	"github.com/ugent-library/bbl/csv"
	"github.com/ugent-library/bbl/oaidc"
)

var config Config

func RunWithContext(ctx context.Context) error {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("bbl")
	v.SetDefault("env", "production")
	v.SetDefault("port", 3000)
	v.SetDefault("s3.region", "us-east-1")
	v.BindEnv("env")
	v.BindEnv("host")
	v.BindEnv("port")
	v.BindEnv("base_url")
	v.BindEnv("pg_conn")
	v.BindEnv("s3.url")
	v.BindEnv("s3.region")
	v.BindEnv("s3.id")
	v.BindEnv("s3.secret")
	v.BindEnv("s3.bucket")
	v.BindEnv("opensearch.url")
	v.BindEnv("opensearch.username")
	v.BindEnv("opensearch.password")
	v.BindEnv("oidc.issuer_url")
	v.BindEnv("oidc.client_id")
	v.BindEnv("oidc.client_secret")
	v.BindEnv("citeproc_url")
	v.BindEnv("secret")
	v.BindEnv("hash_secret")

	if err := v.Unmarshal(&config); err != nil {
		return err
	}

	bbl.RegisterWorkEncoder("oai_dc", oaidc.EncodeWork)
	bbl.RegisterWorkEncoder("mla", csl.NewWorkEncoder(config.CiteprocURL, "mla"))

	bbl.RegisterWorkExporter("csv", csv.NewWorkExporter)

	if err := fang.Execute(ctx, rootCmd); err != nil {
		return err
	}

	return nil
}

func Run() {
	cobra.CheckErr(RunWithContext(context.Background()))
}
