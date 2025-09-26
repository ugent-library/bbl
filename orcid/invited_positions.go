package orcid

func (c *Client) InvitedPositions(id string) (*InvitedPositions, []byte, error) {
	data := &InvitedPositions{}
	b, err := c.get(id+"/invited-positions", data)
	return data, b, err
}
