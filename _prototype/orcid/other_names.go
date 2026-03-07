package orcid

import "context"

func (c *Client) OtherNames(ctx context.Context, id string) (*OtherNames, []byte, error) {
	data := &OtherNames{}
	b, err := c.get(ctx, id+"/other-names", nil, data)
	return data, b, err
}

func (c *Client) OtherName(ctx context.Context, id, putCode string) (*OtherName, []byte, error) {
	data := &OtherName{}
	b, err := c.get(ctx, id+"/other-names/"+putCode, nil, data)
	return data, b, err
}
