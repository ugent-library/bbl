package orcid

import "context"

func (c *Client) Emails(ctx context.Context, id string) (*Emails, []byte, error) {
	data := &Emails{}
	b, err := c.get(ctx, id+"/email", nil, data)
	return data, b, err
}
