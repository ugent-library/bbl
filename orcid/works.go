package orcid

func (c *Client) Works(id string) (*Works, []byte, error) {
	data := &Works{}
	b, err := c.get(id+"/works", data)
	return data, b, err
}
