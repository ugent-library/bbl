package bbl

import "time"

// TODO attr naming, use spec and name like oai?
type Set struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Representation struct {
	WorkID    string    `json:"work_id"`
	Scheme    string    `json:"scheme"`
	Record    []byte    `json:"record"`
	UpdatedAt time.Time `json:"updated_at"`
	Sets      []string  `json:"sets"`
}
