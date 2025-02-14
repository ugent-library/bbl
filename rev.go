package bbl

import (
	"encoding/json"
)

type Op string

const (
	OpAddRec Op = "add_rec"
	// OpSetKind Op = "set_kind"
	OpDelRec  Op = "del_rec"
	OpAddAttr Op = "add_attr"
	OpSetAttr Op = "set_attr"
	OpDelAttr Op = "del_attr"
)

type Rev struct {
	ID      string
	Changes []*Change
}

// TODO split into Change and DbChange
// TODO just flatten?
type Change struct {
	ID   string `json:"id"`
	Op   Op     `json:"op"`
	Args any    `json:"args"`
}

func (c *Change) UnmarshalJSON(b []byte) error {
	rawChange := struct {
		ID   string          `json:"id"`
		Op   Op              `json:"op"`
		Args json.RawMessage `json:"args"`
	}{}
	if err := json.Unmarshal(b, &rawChange); err != nil {
		return err
	}
	var args any
	switch rawChange.Op {
	case OpAddRec:
		args = &AddRecArgs{}
	case OpDelRec:
		args = &DelRecArgs{}
	case OpAddAttr:
		args = &AddAttrArgs{}
	case OpSetAttr:
		args = &SetAttrArgs{}
	case OpDelAttr:
		args = &DelAttrArgs{}
	}
	if err := json.Unmarshal(rawChange.Args, &args); err != nil {
		return err
	}
	*c = Change{
		ID:   rawChange.ID,
		Op:   rawChange.Op,
		Args: args,
	}
	return nil
}

func (c *Change) AddRecArgs() *AddRecArgs { return c.Args.(*AddRecArgs) }

// func (c *Change) SetKindArgs() *SetKindArgs { return c.Args.(*SetKindArgs) }
func (c *Change) DelRecArgs() *DelRecArgs   { return c.Args.(*DelRecArgs) }
func (c *Change) AddAttrArgs() *AddAttrArgs { return c.Args.(*AddAttrArgs) }
func (c *Change) SetAttrArgs() *SetAttrArgs { return c.Args.(*SetAttrArgs) }
func (c *Change) DelAttrArgs() *DelAttrArgs { return c.Args.(*DelAttrArgs) }

type AddRecArgs struct {
	Kind string `json:"kind"`
}

// func AddRec(id, kind string) *Change {
// 	return &Change{ID: id, Op: OpAddRec, Args: &AddRecArgs{Kind: kind}}
// }

// type SetKindArgs struct {
// 	Kind string `json:"kind"`
// }

// func SetKind(id, kind string) *Change {
// 	return &Change{ID: id, Op: OpSetKind, Args: &SetKindArgs{Kind: kind}}
// }

type DelRecArgs struct{}

// func DelRec(id string) *Change {
// 	return &Change{ID: id, Op: OpDelRec, Args: &DelRecArgs{}}
// }

type AddAttrArgs struct {
	ID   string          `json:"id"`
	Kind string          `json:"kind"`
	Val  json.RawMessage `json:"val"`
}

// func AddAttr(id, partID, kind string, val any) *Change {
// 	b, _ := json.Marshal(val)
// 	return &Change{ID: id, Op: OpAddAttr, Args: &AddAttrArgs{ID: partID, Kind: kind, Val: b}}
// }

type SetAttrArgs struct {
	ID string `json:"id"`
	// Kind string          `json:"kind"`
	Val json.RawMessage `json:"val"`
}

// func SetAttr(id, partID string, val any) *Change {
// 	b, _ := json.Marshal(val)
// 	return &Change{ID: id, Op: OpSetAttr, Args: &SetAttrArgs{ID: id, Val: b}}
// }

type DelAttrArgs struct {
	ID string `json:"id"`
	// Kind string `json:"kind"`
}

// func DelAttr(id, partID string) *Change {
// 	return &Change{ID: id, Op: OpDelAttr, Args: &DelAttrArgs{ID: id}}
// }
