package orcid

import "context"

func (c *Client) Memberships(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/memberships", nil, data)
	return data, b, err
}
