package cli

type Config struct {
	Env     string `mapstructure:"env"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	BaseURL string `mapstructure:"base_url"`
	PgConn  string `mapstructure:"pg_conn"`
}
