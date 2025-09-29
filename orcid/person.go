package orcid

import "context"

func (c *Client) Person(ctx context.Context, id string) (*Person, []byte, error) {
	data := &Person{}
	b, err := c.get(ctx, id+"/person", nil, data)
	return data, b, err
}
