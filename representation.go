package bbl

import "time"

type Representation struct {
	WorkID    string    `json:"work_id"`
	Scheme    string    `json:"scheme"`
	Record    []byte    `json:"record"`
	UpdatedAt time.Time `json:"updated_at"`
	Sets      []string  `json:"sets"`
}
