package orcid

import "context"

func (c *Client) Services(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/services", nil, data)
	return data, b, err
}

func (c *Client) Service(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/service/"+putCode, nil, data)
	return data, b, err
}
