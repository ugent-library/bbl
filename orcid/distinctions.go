package orcid

import "context"

func (c *Client) Distinctions(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/distinctions", nil, data)
	return data, b, err
}
