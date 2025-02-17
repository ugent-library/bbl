package bbl

import (
	"encoding/json"
)

type Attr[T any] struct {
	ID  string `json:"id,omitempty"`
	Val T      `json:"val"`
}

func (a *Attr[T]) Set() bool {
	return a != nil
}

func (a *Attr[T]) GetVal() T {
	if a != nil {
		return a.Val
	} else {
		var t T
		return t
	}
}

type RelAttr[T, TT any] struct {
	Attr[T]
	RelID string `json:"rel_id"`
	Rel   TT     `json:"-"`
}

func loadAttr[T any](rec *DbRec, kind string, ptr *Attr[T]) error {
	for _, p := range rec.Attrs {
		if p.Kind == kind {
			attr := Attr[T]{ID: p.ID}
			if err := json.Unmarshal(p.Val, &attr.Val); err != nil {
				return err
			}
			*ptr = attr
			break
		}
	}
	return nil
}

func loadAttrs[T any](rec *DbRec, kind string, ptr *[]Attr[T]) error {
	var attrs []Attr[T]
	for _, p := range rec.Attrs {
		if p.Kind == kind {
			attr := Attr[T]{ID: p.ID}
			if err := json.Unmarshal(p.Val, &attr.Val); err != nil {
				return err
			}
			attrs = append(attrs, attr)
		}
	}
	*ptr = attrs
	return nil
}

func loadRelAttr[T, TT any](rec *DbRec, kind string, ptr *RelAttr[T, TT]) error {
	for _, p := range rec.Attrs {
		if p.Kind == kind {
			var attr RelAttr[T, TT]
			attr.ID = p.ID
			attr.RelID = p.RelID
			if err := json.Unmarshal(p.Val, &attr.Val); err != nil {
				return err
			}
			if p.Rel != nil {
				if err := json.Unmarshal(p.Rel, &attr.Rel); err != nil {
					return err
				}
			}
			*ptr = attr
			break
		}
	}
	return nil
}

func loadRelAttrs[T, TT any](rec *DbRec, kind string, ptr *[]RelAttr[T, TT]) error {
	var attrs []RelAttr[T, TT]
	for _, p := range rec.Attrs {
		if p.Kind == kind {
			var attr RelAttr[T, TT]
			attr.ID = p.ID
			attr.RelID = p.RelID
			if err := json.Unmarshal(p.Val, &attr.Val); err != nil {
				return err
			}
			if p.Rel != nil {
				if err := json.Unmarshal(p.Rel, &attr.Rel); err != nil {
					return err
				}
			}
			attrs = append(attrs, attr)
		}
	}
	*ptr = attrs
	return nil
}
