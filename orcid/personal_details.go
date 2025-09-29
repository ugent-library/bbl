package orcid

import "context"

func (c *Client) PersonalDetails(ctx context.Context, id string) (*PersonalDetails, []byte, error) {
	data := &PersonalDetails{}
	b, err := c.get(ctx, id+"/personal-details", nil, data)
	return data, b, err
}
