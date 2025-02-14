package bbl

import (
	"go.breu.io/ulid"
)

var RecSpecs = map[string]*RecSpec{
	"organization": {
		Attrs: map[string]*AttrSpec{
			"ceased_on": {},
			"name":      {},
		},
	},
}

type RecSpec struct {
	Attrs map[string]*AttrSpec
}

type AttrSpec struct {
}

// type RecSpec[T any] struct {
// 	Attrs map[string]AttrSpec[T]
// }

// type AttrSpec[T any] struct {
// 	Decode func([]byte) (any, error)
// 	Reify  func(T)
// }

// type Rec[T any] struct {
// 	spec    *RecSpec[*Rec[T]]
// 	dbAttrs []*DbAttr
// 	changes []*Change

// 	ID   string `json:"id,omitempty"`
// 	Kind string `json:"kind"`

// 	Attrs T `json:"attrs"`
// }

// func (r *Rec[T]) change(c *Change) (bool, error) {
// 	switch c.Op {
// 	case OpSetKind:
// 		args := c.SetKindArgs()
// 		if args.Kind == r.Kind {
// 			continue
// 		}
// 		r.Kind = args.Kind
// 	case OpAddAttr:
// 		args := c.AddAttrArgs()

// 		if c.ID != r.ID {
// 			return errors.New("id's don't match")
// 		}

// 		attrSpec, ok := r.spec.Attrs[args.Kind]
// 		if !ok {
// 			return errors.New("invalid attr kind")
// 		}

// 		r.dbAttrs = append(r.dbAttrs, &DbAttr{
// 			ID:   args.ID,
// 			Kind: args.Kind,
// 			Val:  args.Val,
// 		})

// 		attrSpec.Reify(r)
// 	case OpSetAttr:
// 		args := c.SetAttrArgs()

// 		if c.ID != r.ID {
// 			return errors.New("id's don't match")
// 		}

// 		attrSpec, ok := r.spec.Attrs[args.Kind]
// 		if !ok {
// 			return errors.New("invalid part kind")
// 		}

// 		var attr *DbAttr
// 		for _, p := range r.dbAttrs {
// 			if args.ID == p.ID {
// 				attr = p
// 				break
// 			}
// 		}

// 		if attr == nil {
// 			return errors.New("part not found")
// 		}

// 		if attr.Kind != args.Kind {
// 			return errors.New("part kind doesn't match")
// 		}

// 		oldVal, err := attrSpec.Decode(attr.Val)
// 		if err != nil {
// 			return err
// 		}
// 		newVal, err := attrSpec.Decode(args.Val)
// 		if err != nil {
// 			return err
// 		}
// 		if reflect.DeepEqual(oldVal, newVal) {
// 			continue
// 		}

// 		attr.Val = args.Val

// 		attrSpec.Reify(r)
// 	case OpDelAttr:
// 		args := c.DelAttrArgs()

// 		if c.ID != r.ID {
// 			return errors.New("id's don't match")
// 		}

// 		attrSpec, ok := r.spec.Attrs[args.Kind]
// 		if !ok {
// 			return errors.New("invalid part kind")
// 		}

// 		var found bool
// 		for i, p := range r.dbAttrs {
// 			if args.ID == p.ID {
// 				if p.Kind != args.Kind {
// 					return errors.New("attr kind doesn't match")
// 				}
// 				found = true
// 				r.dbAttrs = slices.Delete(r.dbAttrs, i, i+1)
// 				break
// 			}
// 		}

// 		if found {
// 			attrSpec.Reify(r)
// 		} else {
// 			continue
// 		}
// 	}

// 	return nil
// }

// func decodeVal[T any](b []byte) (any, error) {
// 	var t T
// 	if err := json.Unmarshal(b, &t); err != nil {
// 		return nil, err
// 	}
// 	return &t, nil
// }

// func setAttrs[T, TT any](rec *Rec[T], kind string, ptr *[]Attr[TT]) {
// 	var attrs []Attr[TT]
// 	for _, p := range rec.dbAttrs {
// 		if p.Kind == kind {
// 			var attr Attr[TT]
// 			json.Unmarshal(p.Val, &attr.Val) // TODO error handling
// 			attrs = append(attrs, attr)
// 		}
// 	}
// 	*ptr = attrs
// }

// func setAttr[T, TT any](rec *Rec[T], kind string, ptr *Attr[TT]) {
// 	for _, p := range rec.dbAttrs {
// 		if p.Kind == kind {
// 			var attr Attr[TT]
// 			json.Unmarshal(p.Val, &attr.Val) // TODO error handling
// 			*ptr = attr
// 			break
// 		}
// 	}
// }

func newID() string {
	return ulid.Make().UUIDString()
}
