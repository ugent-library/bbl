package bbl

import (
	"encoding/json"
)

type Op string

const (
	OpAddRec  Op = "add_rec"
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

func (c *Change) AddRecArgs() *AddRecArgs   { return c.Args.(*AddRecArgs) }
func (c *Change) DelRecArgs() *DelRecArgs   { return c.Args.(*DelRecArgs) }
func (c *Change) AddAttrArgs() *AddAttrArgs { return c.Args.(*AddAttrArgs) }
func (c *Change) SetAttrArgs() *SetAttrArgs { return c.Args.(*SetAttrArgs) }
func (c *Change) DelAttrArgs() *DelAttrArgs { return c.Args.(*DelAttrArgs) }

type AddRecArgs struct {
	Kind string `json:"kind"`
}

type DelRecArgs struct{}

type AddAttrArgs struct {
	ID   string          `json:"id"`
	Kind string          `json:"kind"`
	Val  json.RawMessage `json:"val"`
}

type SetAttrArgs struct {
	ID  string          `json:"id"`
	Val json.RawMessage `json:"val"`
}

type DelAttrArgs struct {
	ID string `json:"id"`
}
