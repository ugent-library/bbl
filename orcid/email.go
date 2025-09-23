package orcid

import "net/http"

func (c *Client) Emails(id string) (*Emails, *http.Response, error) {
	data := &Emails{}
	res, err := c.get(id+"/email", data)
	if err != nil {
		return nil, res, err
	}
	return data, res, nil
}
