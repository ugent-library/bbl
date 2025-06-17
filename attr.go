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

func (c Code) String() string {
	return c.Scheme + ":" + c.Val
}

type Conference struct {
	Name      string    `json:"name,omitempty"`
	Organizer string    `json:"organizer,omitempty"`
	Location  string    `json:"location,omitempty"`
	StartDate time.Time `json:"start_date,omitzero"`
	EndDate   time.Time `json:"end_date,omitzero"`
}

type Extent struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

func (e Extent) String() string {
	return e.Start + "-" + e.End
}

func IsZero[T comparable](t T) bool {
	var tt T
	return t == tt
}
