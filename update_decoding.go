package bbl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeUpdate decodes a JSON-encoded update into a concrete update type.
//
// Wire format:
//
//	{"set": "work:volume", "id": "01J...", "val": "42"}
//	{"hide": "work:volume", "id": "01J..."}
//	{"unset": "work:volume", "id": "01J..."}
//	{"create": "work", "id": "01J...", "kind": "journal_article"}
//	{"delete": "work", "id": "01J..."}
func DecodeUpdate(data []byte) (any, error) {
	var envelope struct {
		Set    string `json:"set"`
		Hide   string `json:"hide"`
		Unset  string `json:"unset"`
		Create string `json:"create"`
		Delete string `json:"delete"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("decode update: %w", err)
	}

	var op, target string
	switch {
	case envelope.Set != "":
		op, target = "set", envelope.Set
	case envelope.Hide != "":
		op, target = "hide", envelope.Hide
	case envelope.Unset != "":
		op, target = "unset", envelope.Unset
	case envelope.Create != "":
		op, target = "create", envelope.Create
	case envelope.Delete != "":
		op, target = "delete", envelope.Delete
	default:
		return nil, fmt.Errorf("decode update: missing operation (set/hide/unset/create/delete)")
	}

	// Lifecycle operations — typed structs.
	if op == "create" || op == "delete" {
		return decodeLifecycle(op, target, data)
	}

	// Field operations — generic Set/Hide/Unset via catalog.
	return decodeFieldOp(op, target, data)
}

// decodeLifecycle handles create/delete with typed structs.
func decodeLifecycle(op, target string, data []byte) (any, error) {
	var m any
	switch op {
	case "create":
		switch target {
		case "work":
			m = &CreateWork{}
		case "person":
			m = &CreatePerson{}
		case "project":
			m = &CreateProject{}
		case "organization":
			m = &CreateOrganization{}
		default:
			return nil, fmt.Errorf("unknown create target %q", target)
		}
	case "delete":
		switch target {
		case "work":
			m = &DeleteWork{}
		case "person":
			m = &DeletePerson{}
		case "project":
			m = &DeleteProject{}
		case "organization":
			m = &DeleteOrganization{}
		default:
			return nil, fmt.Errorf("unknown delete target %q", target)
		}
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("decode %s:%s: %w", op, target, err)
	}
	return m, nil
}

// decodeFieldOp handles set/hide/unset via the field catalog.
// Target format: "entity:field" (e.g. "work:volume", "person:given_name").
func decodeFieldOp(op, target string, data []byte) (any, error) {
	entityType, field, ok := strings.Cut(target, ":")
	if !ok {
		return nil, fmt.Errorf("decode %s: invalid target %q (expected entity:field)", op, target)
	}

	// Validate the field exists in the catalog.
	ft, err := resolveFieldType(entityType, field)
	if err != nil {
		return nil, fmt.Errorf("decode %s %s: %w", op, target, err)
	}

	// Parse the record ID.
	var idHolder struct {
		ID *ID `json:"id"`
	}
	if err := json.Unmarshal(data, &idHolder); err != nil {
		return nil, fmt.Errorf("decode %s %s: %w", op, target, err)
	}
	if idHolder.ID == nil {
		return nil, fmt.Errorf("decode %s %s: missing id", op, target)
	}
	recordID := *idHolder.ID

	switch op {
	case "set":
		var valHolder struct {
			Val json.RawMessage `json:"val"`
		}
		if err := json.Unmarshal(data, &valHolder); err != nil {
			return nil, fmt.Errorf("decode set %s: %w", target, err)
		}
		val, err := ft.unmarshal(valHolder.Val)
		if err != nil {
			return nil, fmt.Errorf("decode set %s: unmarshal val: %w", target, err)
		}
		return &Set{RecordType: entityType, RecordID: recordID, Field: field, Val: val}, nil

	case "hide":
		return &Hide{RecordType: entityType, RecordID: recordID, Field: field}, nil

	case "unset":
		return &Unset{RecordType: entityType, RecordID: recordID, Field: field}, nil
	}

	return nil, fmt.Errorf("unknown operation %q", op)
}
