package orcid

import "context"

func (c *Client) Keywords(ctx context.Context, id string) (*Keywords, []byte, error) {
	data := &Keywords{}
	b, err := c.get(ctx, id+"/keywords", nil, data)
	return data, b, err
}

func (c *Client) Keyword(ctx context.Context, id, putCode string) (*Keyword, []byte, error) {
	data := &Keyword{}
	b, err := c.get(ctx, id+"/keywords/"+putCode, nil, data)
	return data, b, err
}
