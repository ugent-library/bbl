package orcid

import (
	"context"
)

func (c *Client) Activities(ctx context.Context, id string) (*ActivitiesSummary, []byte, error) {
	data := &ActivitiesSummary{}
	b, err := c.get(ctx, id+"/activities", nil, data)
	return data, b, err
}
