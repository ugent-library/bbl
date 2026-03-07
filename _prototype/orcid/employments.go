package orcid

import "context"

func (c *Client) Employments(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/employments", nil, data)
	return data, b, err
}

func (c *Client) Employment(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/employment/"+putCode, nil, data)
	return data, b, err
}
