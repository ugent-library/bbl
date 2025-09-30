package orcid

import "context"

func (c *Client) Addresses(ctx context.Context, id string) (*Addresses, []byte, error) {
	data := &Addresses{}
	b, err := c.get(ctx, id+"/address", nil, data)
	return data, b, err
}

func (c *Client) Address(ctx context.Context, id, putCode string) (*Address, []byte, error) {
	data := &Address{}
	b, err := c.get(ctx, id+"/address/"+putCode, nil, data)
	return data, b, err
}
