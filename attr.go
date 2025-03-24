package bbl

type AttrProfile struct {
	Required bool `json:"required"`
}

type SchemeAttrProfile struct {
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

type Identifier struct {
	Scheme string `json:"scheme"`
	Val    string `json:"val"`
}
