package forms

import "github.com/a-h/templ"

type Profile struct {
	BaseName string     `json:"-"`
	Sections []*Section `json:"sections"`
}

type Section struct {
	BaseName string `json:"-"`
	Name     string `json:"name"`
	Fields   []struct {
		Field string   `json:"field"`
		Only  []string `json:"only"`
	} `json:"fields"`
}

func (s *Section) ID() string {
	return s.BaseName + "-" + s.Name
}

func (s *Section) Anchor() templ.SafeURL {
	return templ.SafeURL("#" + s.BaseName + "-" + s.Name)
}
