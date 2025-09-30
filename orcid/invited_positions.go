package orcid

import "context"

func (c *Client) InvitedPositions(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/invited-positions", nil, data)
	return data, b, err
}

func (c *Client) InvitedPosition(ctx context.Context, id, putCode string) (*Affiliation, []byte, error) {
	data := &Affiliation{}
	b, err := c.get(ctx, id+"/invited-position/"+putCode, nil, data)
	return data, b, err
}
