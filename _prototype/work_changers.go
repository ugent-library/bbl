package bbl

import (
	"fmt"
	"slices"
)

// TODO use mutation terminology?
// TODO validate args passed to Unmarshall
var WorkChangers = map[string]func() WorkChanger{
	"set_kind":           func() WorkChanger { return &WorkSetKind{} },
	"set_identifier":     func() WorkChanger { return &WorkSetIdentifier{} },
	"set_classification": func() WorkChanger { return &WorkSetClassification{} },
	"add_keyword":        func() WorkChanger { return &WorkAddKeyword{} },
	"remove_keyword":     func() WorkChanger { return &WorkRemoveKeyword{} },
}

type WorkChanger interface {
	UnmarshalArgs([]string) error
	Apply(*Work) error
}

type RawWorkChanger struct {
	Name string
	Args []string
}

func LoadWorkChangers(rawChangers []RawWorkChanger) ([]WorkChanger, error) {
	var changers []WorkChanger

	for _, rawC := range rawChangers {
		initC, ok := WorkChangers[rawC.Name]
		if !ok {
			return nil, fmt.Errorf("NewWorkChanger: unknown changer %s", rawC.Name)
		}
		c := initC()
		if err := c.UnmarshalArgs(rawC.Args); err != nil {
			return nil, fmt.Errorf("NewWorkChanger: %s: %w", rawC.Name, err)
		}
		changers = append(changers, c)
	}

	return changers, nil
}

type WorkSetKind struct {
	Kind    string `json:"kind"`
	Subkind string `json:"subkind"`
}

func (c *WorkSetKind) UnmarshalArgs(args []string) error {
	c.Kind = args[0]
	if len(args) > 1 {
		c.Subkind = args[1]
	}
	return nil
}

func (c *WorkSetKind) Apply(rec *Work) error {
	rec.Kind = c.Kind
	rec.Subkind = c.Subkind
	return nil
}

type WorkSetIdentifier Code

func (c *WorkSetIdentifier) UnmarshalArgs(args []string) error {
	c.Scheme = args[0]
	c.Val = args[1]
	return nil
}

func (c *WorkSetIdentifier) Apply(rec *Work) error {
	rec.Identifiers = slices.DeleteFunc(rec.Identifiers, func(iden Code) bool { return iden.Scheme == c.Scheme })
	rec.Identifiers = append(rec.Identifiers, Code{Scheme: c.Scheme, Val: c.Val})
	return nil
}

type WorkSetClassification Code

func (c *WorkSetClassification) UnmarshalArgs(args []string) error {
	c.Scheme = args[0]
	c.Val = args[1]
	return nil
}

func (c *WorkSetClassification) Apply(rec *Work) error {
	rec.Classifications = slices.DeleteFunc(rec.Classifications, func(clas Code) bool { return clas.Scheme == c.Scheme })
	rec.Classifications = append(rec.Classifications, Code{Scheme: c.Scheme, Val: c.Val})
	return nil
}

type WorkAddKeyword struct {
	Val string `json:"val"`
}

func (c *WorkAddKeyword) UnmarshalArgs(args []string) error {
	c.Val = args[0]
	return nil
}

func (c *WorkAddKeyword) Apply(rec *Work) error {
	if !slices.Contains(rec.Keywords, c.Val) {
		rec.Keywords = append(rec.Keywords, c.Val)
	}
	return nil
}

type WorkRemoveKeyword struct {
	Val string `json:"val"`
}

func (c *WorkRemoveKeyword) UnmarshalArgs(args []string) error {
	c.Val = args[0]
	return nil
}

func (c *WorkRemoveKeyword) Apply(rec *Work) error {
	var vals []string
	for _, val := range rec.Keywords {
		if val != c.Val {
			vals = append(vals, val)
		}
	}
	rec.Keywords = vals
	return nil
}
