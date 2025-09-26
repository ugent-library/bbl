package orcid

func (c *Client) Distinctions(id string) (*Distinctions, []byte, error) {
	data := &Distinctions{}
	b, err := c.get(id+"/distinctions", data)
	return data, b, err
}
