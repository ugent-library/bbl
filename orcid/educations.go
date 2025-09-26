package orcid

func (c *Client) Educations(id string) (*Educations, []byte, error) {
	data := &Educations{}
	b, err := c.get(id+"/educations", data)
	return data, b, err
}
