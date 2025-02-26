package bbl

import "time"

type Empty struct{}

type Note struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Text struct {
	Lang string `json:"lang"`
	Text string `json:"text"`
}

type Code struct {
	Scheme string `json:"scheme"`
	Code   string `json:"code"`
}

type Conference struct {
	Name      string     `json:"name"`
	Organizer string     `json:"organizer,omitempty"`
	Location  string     `json:"location,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

func (val Conference) IsBlank() bool {
	return val == Conference{}
}

type Contributor struct {
	CreditRole string `json:"credit_role"`
	Name       string `json:"name"`
}

type NameParts struct {
	GivenName  string `json:"given_name,omitempty"`
	FamilyName string `json:"family_name,omitempty"`
	FullName   string `json:"full_name"`
}
