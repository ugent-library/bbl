package cli

type Config struct {
	Env     string `mapstructure:"env"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	BaseURL string `mapstructure:"base_url"`
	PgConn  string `mapstructure:"pg_conn"`
	S3      struct {
		URL    string `mapstructure:"url"`
		Region string `mapstructure:"region"`
		ID     string `mapstructure:"id"`
		Secret string `mapstructure:"secret"`
		Bucket string `mapstructure:"bucket"`
	} `mapstructure:"s3"`
	OpenSearch struct {
		URL      []string `mapstructure:"url"`
		Username string   `mapstructure:"username"`
		Password string   `mapstructure:"password"`
	} `mapstructure:"opensearch"`
	OIDC struct {
		IssuerURL    string `mapstructure:"issuer_url"`
		ClientID     string `mapstructure:"client_id"`
		ClientSecret string `mapstructure:"client_secret"`
	} `mapstructure:"oidc"`
	CookieSecret     string `mapstructure:"cookie_secret"`
	CookieHashSecret string `mapstructure:"cookie_hash_secret"`
}
