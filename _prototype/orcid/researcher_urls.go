package orcid

import "context"

func (c *Client) ResearcherUrls(ctx context.Context, id string) (*ResearcherUrls, []byte, error) {
	data := &ResearcherUrls{}
	b, err := c.get(ctx, id+"/researcher-urls", nil, data)
	return data, b, err
}

func (c *Client) ResearcherUrl(ctx context.Context, id, putCode string) (*ResearcherUrl, []byte, error) {
	data := &ResearcherUrl{}
	b, err := c.get(ctx, id+"/researcher-urls/"+putCode, nil, data)
	return data, b, err
}
