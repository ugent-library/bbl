package orcid

import "context"

func (c *Client) ExternalIdentifiers(ctx context.Context, id string) (*ExternalIdentifiers, []byte, error) {
	data := &ExternalIdentifiers{}
	b, err := c.get(ctx, id+"/external-identifiers", nil, data)
	return data, b, err
}
