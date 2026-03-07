package orcid

import (
	"context"
)

func (c *Client) Record(ctx context.Context, id string) (*Record, []byte, error) {
	data := &Record{}
	b, err := c.get(ctx, id+"/record", nil, data)
	return data, b, err
}
