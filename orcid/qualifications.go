package orcid

import "context"

func (c *Client) Qualifications(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/qualifications", nil, data)
	return data, b, err
}

func (c *Client) Qualification(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/qualification/"+putCode, nil, data)
	return data, b, err
}
