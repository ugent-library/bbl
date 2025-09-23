package orcid

func (c *Client) Keywords(id string) (*Keywords, []byte, error) {
	data := &Keywords{}
	b, err := c.get(id+"/keywords", data)
	return data, b, err
}
