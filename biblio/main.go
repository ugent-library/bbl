package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/biblio/plato"
	"github.com/ugent-library/bbl/cmd/bbl/cli"
	"github.com/ugent-library/bbl/ldap"
)

func main() {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.BindEnv("ldap.url")
	v.BindEnv("ldap.username")
	v.BindEnv("ldap.password")
	v.BindEnv("plato.url")
	v.BindEnv("plato.username")
	v.BindEnv("plato.password")

	ldapUserSource, err := ldap.New(ldap.Config{
		URL:      v.GetString("ldap.url"),
		Username: v.GetString("ldap.username"),
		Password: v.GetString("ldap.password"),
		Filter:   "(|(objectclass=ugentEmployee)(objectclass=uzEmployee)(objectclass=ugentFormerEmployee)(objectclass=ugentSenior)(objectclass=ugentStudent)(objectclass=ugentUCTStudent)(objectclass=ugentExCoStudent)(objectclass=ugentFormerStudent)(ugentextcategorycode=alum))",
		Attrs: []string{
			"displayName",
			"mail",
			"ugentPersonID",
			"uid",
		},
		MatchIdentifierScheme: "ugentPersonID",
		MappingFunc: func(m map[string][]string) (*bbl.User, error) {
			user := &bbl.User{}
			return user, nil
		},
	})
	cobra.CheckErr(err)

	platoWorkSource, err := plato.New(plato.Config{
		URL:      v.GetString("plato.url"),
		Username: v.GetString("plato.username"),
		Password: v.GetString("plato.password"),
	})
	cobra.CheckErr(err)

	bbl.RegisterUserSource("ldap", ldapUserSource)
	bbl.RegisterWorkSource("plato", platoWorkSource)

	cli.Run()
}
