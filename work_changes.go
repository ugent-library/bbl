package bbl

import (
	"slices"
)

var WorkChanges = map[string]func() WorkChange{
	"set_kind":           func() WorkChange { return &WorkSetKind{} },
	"set_identifier":     func() WorkChange { return &WorkSetIdentifier{} },
	"set_classification": func() WorkChange { return &WorkSetClassification{} },
	"add_keyword":        func() WorkChange { return &WorkAddKeyword{} },
	"remove_keyword":     func() WorkChange { return &WorkRemoveKeyword{} },
}

type WorkChange interface {
	UnmarshalArgs([]string) error
	Apply(*Work) error
}

type WorkSetKind struct {
	Kind    string `json:"kind"`
	Subkind string `json:"subkind"`
}

func (c *WorkSetKind) UnmarshalArgs(args []string) error {
	c.Kind = args[0]
	c.Subkind = args[1]
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
