package orcid

import "context"

func (c *Client) PeerReviews(ctx context.Context, id string) (*PeerReviews, []byte, error) {
	data := &PeerReviews{}
	b, err := c.get(ctx, id+"/peer-reviews", nil, data)
	return data, b, err
}
