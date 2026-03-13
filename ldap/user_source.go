package ldap

import (
	"context"
	"iter"

	ldaplib "github.com/go-ldap/ldap/v3"
	"github.com/ugent-library/bbl"
)

// Mapping declares which LDAP attribute name feeds each ImportUserInput field.
// Attrs for the search request are derived automatically from non-empty fields.
// Identifiers maps identifier scheme names to LDAP attribute names,
// e.g. {"ugent_id": "ugentID", "orcid": "ORCID-Number"}.
type Mapping struct {
	SourceID    string            `yaml:"source_id"`
	Username    string            `yaml:"username"`
	Email       string            `yaml:"email"`
	Name        string            `yaml:"name"`
	Role        string            `yaml:"role"`
	Identifiers map[string]string `yaml:"identifiers"` // scheme → attr
}

// Config holds connection, search, and field-mapping parameters for an LDAP directory.
type Config struct {
	URL      string  `yaml:"url"`
	Username string  `yaml:"username"`
	Password string  `yaml:"password"`
	Base     string  `yaml:"base"`
	Filter   string  `yaml:"filter"`
	Mapping  Mapping `yaml:"mapping"`
}

// UserSource streams user records from an LDAP directory.
type UserSource struct {
	config Config
}

// New returns a UserSource for the given Config.
func New(c Config) *UserSource {
	return &UserSource{config: c}
}

// attrs returns the deduplicated list of LDAP attribute names needed by the mapping.
func (s *UserSource) attrs() []string {
	m := s.config.Mapping
	seen := make(map[string]bool)
	var attrs []string
	for _, a := range []string{m.SourceID, m.Username, m.Email, m.Name, m.Role} {
		if a != "" && !seen[a] {
			seen[a] = true
			attrs = append(attrs, a)
		}
	}
	for _, a := range m.Identifiers {
		if !seen[a] {
			seen[a] = true
			attrs = append(attrs, a)
		}
	}
	return attrs
}

// mapEntry translates an LDAP attribute map into an ImportUserInput using the
// declared field mapping. Missing attributes silently produce empty strings.
func (s *UserSource) mapEntry(raw map[string][]string) *bbl.ImportUserInput {
	m := s.config.Mapping
	in := &bbl.ImportUserInput{
		SourceID: first(raw[m.SourceID]),
		Username: first(raw[m.Username]),
		Email:    first(raw[m.Email]),
		Name:     first(raw[m.Name]),
		Role:     first(raw[m.Role]),
	}
	for scheme, attr := range m.Identifiers {
		for _, val := range raw[attr] {
			if val != "" {
				in.Identifiers = append(in.Identifiers, bbl.UserIdentifier{Scheme: scheme, Val: val})
			}
		}
	}
	return in
}

func first(vals []string) string {
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Iter connects to the LDAP server and returns an iterator over import records.
// Connection and bind errors are returned immediately as the second return value.
// Per-entry errors are yielded inline. The connection is closed when iteration
// ends or is broken.
func (s *UserSource) Iter(ctx context.Context) (iter.Seq2[*bbl.ImportUserInput, error], error) {
	conn, err := ldaplib.DialURL(s.config.URL)
	if err != nil {
		return nil, err
	}

	if err = conn.Bind(s.config.Username, s.config.Password); err != nil {
		conn.Close()
		return nil, err
	}

	attrs := s.attrs()

	seq := func(yield func(*bbl.ImportUserInput, error) bool) {
		defer conn.Close()

		req := ldaplib.NewSearchRequest(
			s.config.Base,
			ldaplib.ScopeWholeSubtree,
			ldaplib.NeverDerefAliases,
			0, 0, false,
			s.config.Filter,
			attrs,
			nil,
		)

		res := conn.SearchAsync(ctx, req, 2000)

		for res.Next() {
			entry := res.Entry()
			raw := make(map[string][]string, len(attrs))
			for _, attr := range attrs {
				raw[attr] = entry.GetAttributeValues(attr)
			}
			if !yield(s.mapEntry(raw), nil) {
				return
			}
		}

		if err := res.Err(); err != nil {
			yield(nil, err)
		}
	}

	return seq, nil
}
