package bbl

import "time"

type AttrProfile struct {
	Required bool `json:"required"`
}

type CodeAttrProfile struct {
	AttrProfile
	Schemes []struct {
		Scheme   string `json:"scheme"`
		Required bool   `json:"required"`
	} `json:"schemes"`
}

type Text struct {
	Lang string `json:"lang"`
	Val  string `json:"val"`
}

type Code struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
}

type Conference struct {
	Name      string    `json:"name,omitempty"`
	Organizer string    `json:"organizer,omitempty"`
	Location  string    `json:"location,omitempty"`
	StartDate time.Time `json:"start_date,omitzero"`
	EndDate   time.Time `json:"end_date,omitzero"`
}
