package bbl

import "time"

type Empty struct{}

type Text struct {
	Lang string `json:"lang"`
	Text string `json:"text"`
}

type Identifier struct {
	Scheme string `json:"scheme"`
	Value  string `json:"value"`
}

type Conference struct {
	Name      string     `json:"name"`
	Organizer string     `json:"organizer,omitempty"`
	Location  string     `json:"location,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

type RelatedProject struct{}
