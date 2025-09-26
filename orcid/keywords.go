package orcid

import "context"

func (c *Client) Keywords(ctx context.Context, id string) (*Keywords, []byte, error) {
	data := &Keywords{}
	b, err := c.get(ctx, id+"/keywords", nil, data)
	return data, b, err
}
