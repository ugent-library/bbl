package ldap

import (
	"context"
	"iter"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ugent-library/bbl"
)

type UserSource struct {
	config Config
}

type Config struct {
	URL                   string
	Username              string
	Password              string
	Base                  string
	Filter                string
	Attrs                 []string
	MappingFunc           func(map[string][]string) (*bbl.User, error)
	MatchIdentifierScheme string
}

func New(c Config) (*UserSource, error) {
	return &UserSource{
		config: c,
	}, nil
}

func (us *UserSource) Interval() time.Duration {
	return 24 * time.Hour
}

func (us *UserSource) MatchIdentifierScheme() string {
	return us.config.MatchIdentifierScheme
}

func (us *UserSource) Iter(ctx context.Context) (iter.Seq[*bbl.User], func() error) {
	var iterErr error

	finish := func() error { return iterErr }

	seq := func(yield func(*bbl.User) bool) {
		conn, err := ldap.DialURL(us.config.URL)
		if err != nil {
			iterErr = err
			return
		}
		defer conn.Close()

		if err = conn.Bind(us.config.Username, us.config.Password); err != nil {
			iterErr = err
			return
		}

		req := ldap.NewSearchRequest(
			us.config.Base,
			ldap.ScopeSingleLevel,
			ldap.NeverDerefAliases,
			0, 0, false,
			us.config.Filter,
			us.config.Attrs,
			[]ldap.Control{},
		)

		res := conn.SearchAsync(ctx, req, 2000)

		for res.Next() {
			entry := res.Entry()
			attrs := make(map[string][]string)

			for _, attr := range us.config.Attrs {
				attrs[attr] = entry.GetAttributeValues(attr)
			}

			user, err := us.config.MappingFunc(attrs)
			if err != nil {
				iterErr = err
				return
			}
			if !yield(user) {
				return
			}
		}
	}

	return seq, finish
}
