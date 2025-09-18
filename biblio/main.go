package main

import (
	"strings"
	"time"

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
	v.BindEnv("ldap.base")
	v.BindEnv("ldap.filter")
	v.BindEnv("plato.url")
	v.BindEnv("plato.username")
	v.BindEnv("plato.password")

	ldapUserSource, err := ldap.New(ldap.Config{
		URL:      v.GetString("ldap.url"),
		Username: v.GetString("ldap.username"),
		Password: v.GetString("ldap.password"),
		Base:     v.GetString("ldap.base"),
		Filter:   v.GetString("ldap.filter"),
		Attrs: []string{
			"displayName",
			"mail",
			"ugentIDs",
			"ugentPersonID",
			"uid",
		},
		MatchIdentifierScheme: "ugent_person_id",
		MappingFunc: func(m map[string][]string) (*bbl.User, error) {
			rec := &bbl.User{
				DeactivateAt: time.Now().Add(time.Hour * 24 * 30),
			}

			if vals := m["ugentPersonID"]; len(vals) != 0 {
				ident := bbl.Code{Scheme: "ugent_person_id", Val: vals[0]}
				rec.ID = ident.String()
				rec.Identifiers = append(rec.Identifiers, ident)
			}
			for _, val := range m["ugentIDs"] {
				rec.Identifiers = append(rec.Identifiers, bbl.Code{Scheme: "ugent_id", Val: val})
			}
			if vals := m["uid"]; len(vals) != 0 {
				rec.Username = vals[0]
			}
			if vals := m["displayName"]; len(vals) != 0 {
				rec.Name = vals[0]
			}
			if vals := m["mail"]; len(vals) != 0 {
				rec.Email = vals[0]
			}

			return rec, nil
		},
	})
	cobra.CheckErr(err)

	platoWorkSource, err := plato.New(plato.Config{
		URL:      v.GetString("plato.url"),
		Username: v.GetString("plato.username"),
		Password: v.GetString("plato.password"),
	})
	cobra.CheckErr(err)

	bbl.RegisterUserSource("ugent_ldap", ldapUserSource)
	bbl.RegisterWorkSource("plato", platoWorkSource)

	cli.Run()
}
