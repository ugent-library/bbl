package ldap

import (
	"context"
	"iter"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/ugent-library/bbl"
)

type UserSource struct {
	conn                  *ldap.Conn
	base                  string
	filter                string
	attrs                 []string
	mappingFunc           func(map[string][]string) (*bbl.User, error)
	matchIdentifierScheme string
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

// TODO conn.Close()
func New(c Config) (*UserSource, error) {
	conn, err := ldap.DialURL(c.URL)
	if err != nil {
		return nil, err
	}

	if err = conn.Bind(c.Username, c.Password); err != nil {
		return nil, err
	}

	return &UserSource{
		conn:                  conn,
		base:                  c.Base,
		filter:                c.Filter,
		attrs:                 c.Attrs,
		matchIdentifierScheme: c.MatchIdentifierScheme,
		mappingFunc:           c.MappingFunc,
	}, nil
}

func (us *UserSource) Interval() time.Duration {
	return 24 * time.Hour
}

func (us *UserSource) MatchIdentifierScheme() string {
	return us.matchIdentifierScheme
}

func (us *UserSource) Iter(ctx context.Context) (iter.Seq[*bbl.User], func() error) {
	var iterErr error

	finish := func() error { return iterErr }

	seq := func(yield func(*bbl.User) bool) {
		req := ldap.NewSearchRequest(
			us.base,
			ldap.ScopeSingleLevel,
			ldap.NeverDerefAliases,
			0, 0, false,
			us.filter,
			us.attrs,
			[]ldap.Control{},
		)

		res := us.conn.SearchAsync(ctx, req, 2000)

		for res.Next() {
			entry := res.Entry()
			attrs := make(map[string][]string)

			for _, attr := range us.attrs {
				attrs[attr] = entry.GetAttributeValues(attr)
			}

			user, err := us.mappingFunc(attrs)
			if err != nil {
				iterErr = err
				break
			}
			if !yield(user) {
				break
			}
		}
	}

	return seq, finish
}
