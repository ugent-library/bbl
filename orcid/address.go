package orcid

func (c *Client) Address(id string) (*Addresses, []byte, error) {
	data := &Addresses{}
	b, err := c.get(id+"/address", data)
	return data, b, err
}
