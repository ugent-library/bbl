package orcid

import "context"

func (c *Client) Biography(ctx context.Context, id string) (*Biography, []byte, error) {
	data := &Biography{}
	b, err := c.get(ctx, id+"/biography", nil, data)
	return data, b, err
}
