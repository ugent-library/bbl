package orcid

func (c *Client) Qualifications(id string) (*Qualifications, []byte, error) {
	data := &Qualifications{}
	b, err := c.get(id+"/qualifications", data)
	return data, b, err
}
