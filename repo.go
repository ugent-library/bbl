package bbl

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.breu.io/ulid"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("version conflict")

func NewID() string {
	return ulid.Make().UUIDString()
}

type Header struct {
	ID          string    `json:"id,omitempty"`
	Version     int       `json:"version,omitzero"`
	CreatedAt   time.Time `json:"created_at,omitzero"`
	UpdatedAt   time.Time `json:"updated_at,omitzero"`
	CreatedByID string    `json:"created_by_id,omitempty"`
	CreatedBy   *User     `json:"created_by,omitempty"`
	UpdatedByID string    `json:"updated_by_id,omitempty"`
	UpdatedBy   *User     `json:"updated_by,omitempty"`
}

func (h *Header) GetHeader() *Header {
	return h
}

type Rec interface {
	GetHeader() *Header
}

type Rev struct {
	UserID  string
	Actions []Action
}

func (r *Rev) Add(actions ...Action) {
	r.Actions = append(r.Actions, actions...)
}

func (r *Rev) UnmarshalJSON(b []byte) error {
	rawRev := struct {
		UserID  string `json:"user_id"`
		Actions []struct {
			Action string          `json:"action"`
			Data   json.RawMessage `json:"data"`
		} `json:"actions"`
	}{}
	if err := json.Unmarshal(b, &rawRev); err != nil {
		return err
	}

	rev := Rev{UserID: rawRev.UserID}

	for _, rawAction := range rawRev.Actions {
		var action Action
		switch rawAction.Action {
		case "create_organization":
			action = &CreateOrganization{}
		case "update_organization":
			action = &UpdateOrganization{}
		case "create_person":
			action = &CreatePerson{}
		case "update_person":
			action = &UpdatePerson{}
		case "create_project":
			action = &CreateProject{}
		case "update_project":
			action = &UpdateProject{}
		case "create_work":
			action = &CreateWork{}
		case "update_work":
			action = &UpdateWork{}
		case "change_work":
			action = &ChangeWork{}
		default:
			return fmt.Errorf("Rev: invalid action %q", rawAction.Action)
		}
		if err := json.Unmarshal(rawAction.Data, action); err != nil {
			return err
		}
		rev.Actions = append(rev.Actions, action)
	}

	*r = rev

	return nil
}

type Action interface {
	isAction()
}

type CreateOrganization struct {
	Organization *Organization `json:"organization"`
}

func (*CreateOrganization) isAction() {}

type UpdateOrganization struct {
	Organization *Organization `json:"organization"`
	MatchVersion bool
}

func (*UpdateOrganization) isAction() {}

type CreatePerson struct {
	Person *Person `json:"person"`
}

func (*CreatePerson) isAction() {}

type UpdatePerson struct {
	Person       *Person `json:"person"`
	MatchVersion bool
}

func (*UpdatePerson) isAction() {}

type CreateProject struct {
	Project *Project `json:"project"`
}

func (*CreateProject) isAction() {}

type UpdateProject struct {
	Project      *Project `json:"project"`
	MatchVersion bool
}

func (*UpdateProject) isAction() {}

type CreateWork struct {
	Work *Work `json:"work"`
}

func (*CreateWork) isAction() {}

type UpdateWork struct {
	Work         *Work `json:"work"`
	MatchVersion bool
}

func (*UpdateWork) isAction() {}

type ChangeWork struct {
	WorkID  string
	Changes []WorkChanger
}

func (*ChangeWork) isAction() {}

func (a *ChangeWork) UnmarshalJSON(b []byte) error {
	rawAction := struct {
		WorkID  string `json:"work_id"`
		Changes []struct {
			Change string          `json:"change"`
			Data   json.RawMessage `json:"data"`
		} `json:"changes"`
	}{}
	if err := json.Unmarshal(b, &rawAction); err != nil {
		return err
	}

	action := ChangeWork{WorkID: rawAction.WorkID}

	for _, rawChange := range rawAction.Changes {
		initChange, ok := WorkChangers[rawChange.Change]
		if !ok {
			return fmt.Errorf("Rev: invalid change %q", rawChange.Change)
		}
		c := initChange()
		if err := json.Unmarshal(rawChange.Data, c); err != nil {
			return fmt.Errorf("Rev: %w", err)
		}
		action.Changes = append(action.Changes, c)
	}

	*a = action

	return nil
}

type GetRepresentationsOpts struct {
	WorkID       string
	Scheme       string
	Limit        int
	UpdatedAtLTE time.Time
	UpdatedAtGTE time.Time
}
