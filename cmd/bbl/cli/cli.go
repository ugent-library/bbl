package cli

import (
	"context"
	"os"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/arxiv"
	"github.com/ugent-library/bbl/csl"
	"github.com/ugent-library/bbl/csv"
	"github.com/ugent-library/bbl/oaidc"
)

var config Config

func RunWithContext(ctx context.Context) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("bbl")
	v.SetDefault("env", "production")
	v.SetDefault("port", 3000)
	v.SetDefault("s3.region", "us-east-1")
	v.BindEnv("base_url")
	v.BindEnv("centrifuge.api.key")
	v.BindEnv("centrifuge.api.url")
	v.BindEnv("centrifuge.transport.url")
	v.BindEnv("centrifuge.hmac_secret")
	v.BindEnv("citeproc_url")
	v.BindEnv("env")
	v.BindEnv("hash_secret")
	v.BindEnv("host")
	v.BindEnv("oidc.client_id")
	v.BindEnv("oidc.client_secret")
	v.BindEnv("oidc.issuer_url")
	v.BindEnv("opensearch.password")
	v.BindEnv("opensearch.url")
	v.BindEnv("opensearch.username")
	v.BindEnv("pg_conn")
	v.BindEnv("port")
	v.BindEnv("s3.bucket")
	v.BindEnv("s3.id")
	v.BindEnv("s3.region")
	v.BindEnv("s3.secret")
	v.BindEnv("s3.url")
	v.BindEnv("secret")

	if err := v.Unmarshal(&config); err != nil {
		cobra.CheckErr(err)
	}

	bbl.RegisterWorkEncoder("oai_dc", oaidc.EncodeWork)
	bbl.RegisterWorkEncoder("mla", csl.NewWorkEncoder(config.CiteprocURL, "mla"))
	bbl.RegisterWorkImporter("arxiv", arxiv.NewWorkImporter())
	bbl.RegisterWorkExporter("csv", csv.NewWorkExporter)

	if err := fang.Execute(ctx, rootCmd); err != nil {
		os.Exit(1)
	}
}

func Run() {
	RunWithContext(context.Background())
}
