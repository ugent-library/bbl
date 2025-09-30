package orcid

import (
	"context"
)

func (c *Client) ResearchResources(ctx context.Context, id string) (*ResearchResources, []byte, error) {
	data := &ResearchResources{}
	b, err := c.get(ctx, id+"/research-resources", nil, data)
	return data, b, err
}
