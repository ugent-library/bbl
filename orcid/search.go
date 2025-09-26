package orcid

import (
	"context"
	"net/url"
)

func (c *Client) Search(ctx context.Context, q string) (*Search, []byte, error) {
	data := &Search{}
	b, err := c.get(ctx, "search", url.Values{"q": []string{q}}, data)
	return data, b, err
}
