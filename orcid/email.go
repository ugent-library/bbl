package orcid

func (c *Client) Emails(id string) (*Emails, []byte, error) {
	data := &Emails{}
	b, err := c.get(id+"/email", data)
	return data, b, err
}
