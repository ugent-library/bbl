package orcid

import "context"

func (c *Client) InvitedPositions(ctx context.Context, id string) (*Affiliations, []byte, error) {
	data := &Affiliations{}
	b, err := c.get(ctx, id+"/invited-positions", nil, data)
	return data, b, err
}
