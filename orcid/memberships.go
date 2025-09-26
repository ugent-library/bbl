package orcid

func (c *Client) Memberships(id string) (*Memberships, []byte, error) {
	data := &Memberships{}
	b, err := c.get(id+"/memberships", data)
	return data, b, err
}
