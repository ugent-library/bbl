package orcid

import "context"

func (c *Client) Distinctions(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/distinctions", nil, data)
	return data, b, err
}

func (c *Client) Distinction(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/distinction/"+putCode, nil, data)
	return data, b, err
}
