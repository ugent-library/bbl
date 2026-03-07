package orcid

import "context"

func (c *Client) Memberships(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/memberships", nil, data)
	return data, b, err
}

func (c *Client) Membership(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/membership/"+putCode, nil, data)
	return data, b, err
}
