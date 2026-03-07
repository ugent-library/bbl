package orcid

import (
	"context"
	"strings"
)

func (c *Client) BulkWorks(ctx context.Context, id string, putCodes []string) (*Bulk, []byte, error) {
	data := &Bulk{}
	b, err := c.get(ctx, id+"/works/"+strings.Join(putCodes, ","), nil, data)
	return data, b, err
}

func (c *Client) Works(ctx context.Context, id string, putCodes ...string) (*Works, []byte, error) {
	data := &Works{}
	b, err := c.get(ctx, id+"/works", nil, data)
	return data, b, err
}

func (c *Client) Work(ctx context.Context, id, putCode string) (*Work, []byte, error) {
	data := &Work{}
	b, err := c.get(ctx, id+"/work/"+putCode, nil, data)
	return data, b, err
}
