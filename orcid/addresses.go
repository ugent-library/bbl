package orcid

import "context"

func (c *Client) Addresses(ctx context.Context, id string) (*Addresses, []byte, error) {
	data := &Addresses{}
	b, err := c.get(ctx, id+"/address", nil, data)
	return data, b, err
}
