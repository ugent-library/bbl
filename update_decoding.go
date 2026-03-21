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
//	{"set": "work_volume", "work_id": "01J...", "val": "42"}
//	{"hide": "work_volume", "work_id": "01J..."}
//	{"unset": "work_volume", "work_id": "01J..."}
//	{"create": "work", "work_id": "01J...", "kind": "journal_article"}
//	{"delete": "work", "work_id": "01J..."}
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
	key := op + ":" + target
	var m any
	switch key {
	case "create:work":
		m = &CreateWork{}
	case "delete:work":
		m = &DeleteWork{}
	case "create:person":
		m = &CreatePerson{}
	case "delete:person":
		m = &DeletePerson{}
	case "create:project":
		m = &CreateProject{}
	case "delete:project":
		m = &DeleteProject{}
	case "create:organization":
		m = &CreateOrganization{}
	case "delete:organization":
		m = &DeleteOrganization{}
	default:
		return nil, fmt.Errorf("unknown lifecycle operation %q", key)
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("decode %s: %w", key, err)
	}
	return m, nil
}

// decodeFieldOp handles set/hide/unset via the field catalog.
// Target format: "entity_field" (e.g. "work_volume", "person_given_name").
func decodeFieldOp(op, target string, data []byte) (any, error) {
	entityType, field, err := splitTarget(target)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", op, err)
	}

	// Validate the field exists in the catalog.
	ft, err := resolveFieldType(entityType, field)
	if err != nil {
		return nil, fmt.Errorf("decode %s:%s: %w", op, target, err)
	}

	// Parse the entity ID.
	var idHolder struct {
		WorkID         *ID `json:"work_id"`
		PersonID       *ID `json:"person_id"`
		ProjectID      *ID `json:"project_id"`
		OrganizationID *ID `json:"organization_id"`
	}
	if err := json.Unmarshal(data, &idHolder); err != nil {
		return nil, fmt.Errorf("decode %s:%s: %w", op, target, err)
	}
	var recordID ID
	switch entityType {
	case "work":
		if idHolder.WorkID == nil {
			return nil, fmt.Errorf("decode %s:%s: missing work_id", op, target)
		}
		recordID = *idHolder.WorkID
	case "person":
		if idHolder.PersonID == nil {
			return nil, fmt.Errorf("decode %s:%s: missing person_id", op, target)
		}
		recordID = *idHolder.PersonID
	case "project":
		if idHolder.ProjectID == nil {
			return nil, fmt.Errorf("decode %s:%s: missing project_id", op, target)
		}
		recordID = *idHolder.ProjectID
	case "organization":
		if idHolder.OrganizationID == nil {
			return nil, fmt.Errorf("decode %s:%s: missing organization_id", op, target)
		}
		recordID = *idHolder.OrganizationID
	}

	switch op {
	case "set":
		// Parse val using the fieldType's unmarshal.
		var valHolder struct {
			Val json.RawMessage `json:"val"`
		}
		if err := json.Unmarshal(data, &valHolder); err != nil {
			return nil, fmt.Errorf("decode set:%s: %w", target, err)
		}
		val, err := ft.unmarshal(valHolder.Val)
		if err != nil {
			return nil, fmt.Errorf("decode set:%s: unmarshal val: %w", target, err)
		}
		return &Set{RecordType: entityType, RecordID: recordID, Field: field, Val: val}, nil

	case "hide":
		return &Hide{RecordType: entityType, RecordID: recordID, Field: field}, nil

	case "unset":
		return &Unset{RecordType: entityType, RecordID: recordID, Field: field}, nil
	}

	return nil, fmt.Errorf("unknown operation %q", op)
}

// splitTarget splits "entity_field" into entity type and field name.
// Entity type is always the first component before the first underscore
// that matches a known entity type.
func splitTarget(target string) (string, string, error) {
	for _, prefix := range []string{"organization_", "project_", "person_", "work_"} {
		if strings.HasPrefix(target, prefix) {
			return prefix[:len(prefix)-1], target[len(prefix):], nil
		}
	}
	return "", "", fmt.Errorf("cannot parse entity type from %q", target)
}
