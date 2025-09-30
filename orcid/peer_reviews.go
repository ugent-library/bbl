package orcid

import "context"

func (c *Client) PeerReviews(ctx context.Context, id string) (*PeerReviews, []byte, error) {
	data := &PeerReviews{}
	b, err := c.get(ctx, id+"/peer-reviews", nil, data)
	return data, b, err
}

func (c *Client) PeerReview(ctx context.Context, id, putCode string) (*PeerReview, []byte, error) {
	data := &PeerReview{}
	b, err := c.get(ctx, id+"/peer-review/"+putCode, nil, data)
	return data, b, err
}
