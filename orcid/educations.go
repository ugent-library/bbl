package orcid

import "context"

func (c *Client) Educations(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/educations", nil, data)
	return data, b, err
}

func (c *Client) Education(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/education/"+putCode, nil, data)
	return data, b, err
}
