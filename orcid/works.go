package orcid

import "context"

func (c *Client) Works(ctx context.Context, id string) (*Works, []byte, error) {
	data := &Works{}
	b, err := c.get(ctx, id+"/works", nil, data)
	return data, b, err
}
