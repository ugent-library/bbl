package orcid

import "context"

func (c *Client) Employments(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/employments", nil, data)
	return data, b, err
}
