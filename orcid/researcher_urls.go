package orcid

import "context"

func (c *Client) ResearcherUrls(ctx context.Context, id string) (*ResearcherUrls, []byte, error) {
	data := &ResearcherUrls{}
	b, err := c.get(ctx, id+"/researcher-urls", nil, data)
	return data, b, err
}
