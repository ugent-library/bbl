package orcid

import "context"

func (c *Client) ExternalIdentifiers(ctx context.Context, id string) (*ExternalIdentifiers, []byte, error) {
	data := &ExternalIdentifiers{}
	b, err := c.get(ctx, id+"/external-identifiers", nil, data)
	return data, b, err
}

func (c *Client) ExternalIdentifier(ctx context.Context, id, putCode string) (*ExternalIdentifier, []byte, error) {
	data := &ExternalIdentifier{}
	b, err := c.get(ctx, id+"/external-identifiers/"+putCode, nil, data)
	return data, b, err
}
