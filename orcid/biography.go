package orcid

func (c *Client) Biography(id string) (*Biography, []byte, error) {
	data := &Biography{}
	b, err := c.get(id+"/biography", data)
	return data, b, err
}
