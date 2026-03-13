package citeformat

import "github.com/ugent-library/bbl"

// Config is the YAML configuration for a citeproc citation style.
type Config struct {
	URL   string `yaml:"url"`   // citeproc-js-server URL
	Style string `yaml:"style"` // CSL style name (e.g. "apa")
}

// WorkEncoder formats works as citations using a citeproc-js-server.
// It implements bbl.WorkEncoder.
type WorkEncoder struct {
	client *Client
}

// New creates a WorkEncoder from a Config.
func New(c Config) (bbl.WorkEncoder, error) {
	return &WorkEncoder{
		client: &Client{URL: c.URL, Style: c.Style},
	}, nil
}

func (e *WorkEncoder) Encode(work *bbl.Work) ([]byte, error) {
	s, err := e.client.Format(work)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

var _ bbl.WorkEncoder = (*WorkEncoder)(nil)
