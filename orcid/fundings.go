package orcid

import (
	"context"
)

func (c *Client) Fundings(ctx context.Context, id string) (*Fundings, []byte, error) {
	data := &Fundings{}
	b, err := c.get(ctx, id+"/fundings", nil, data)
	return data, b, err
}

func (c *Client) Funding(ctx context.Context, id, putCode string) (*Funding, []byte, error) {
	data := &Funding{}
	b, err := c.get(ctx, id+"/funding/"+putCode, nil, data)
	return data, b, err
}
