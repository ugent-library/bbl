package orcid

func (c *Client) Services(id string) (*Services, []byte, error) {
	data := &Services{}
	b, err := c.get(id+"/services", data)
	return data, b, err
}
