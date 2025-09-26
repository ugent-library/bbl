package orcid

import "context"

func (c *Client) Educations(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/educations", nil, data)
	return data, b, err
}
