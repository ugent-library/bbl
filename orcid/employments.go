package orcid

func (c *Client) Employments(id string) (*Employments, []byte, error) {
	data := &Employments{}
	b, err := c.get(id+"/employments", data)
	return data, b, err
}
