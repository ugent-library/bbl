package bbl

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.breu.io/ulid"
)

var ErrNotFound = errors.New("not found")

func NewID() string {
	return ulid.Make().UUIDString()
}

type GetWorkRepresentationsOpts struct {
	WorkID       string
	Scheme       string
	Limit        int
	UpdatedAtLTE time.Time
	UpdatedAtGTE time.Time
}

type Rec interface {
	RecID() string
}

type Rev struct {
	Actions []Action
}

func NewRev() *Rev {
	return &Rev{}
}

func (r *Rev) Add(action Action) {
	r.Actions = append(r.Actions, action)
}

func (r *Rev) UnmarshalJSON(b []byte) error {
	rawRev := struct {
		Actions []struct {
			Action string          `json:"action"`
			Data   json.RawMessage `json:"data"`
		} `json:"actions"`
	}{}
	if err := json.Unmarshal(b, &rawRev); err != nil {
		return err
	}

	rev := Rev{}

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
}

func (*UpdateOrganization) isAction() {}

type CreatePerson struct {
	Person *Person `json:"person"`
}

func (*CreatePerson) isAction() {}

type UpdatePerson struct {
	Person *Person `json:"person"`
}

func (*UpdatePerson) isAction() {}

type CreateProject struct {
	Project *Project `json:"project"`
}

func (*CreateProject) isAction() {}

type UpdateProject struct {
	Project *Project `json:"project"`
}

func (*UpdateProject) isAction() {}

type CreateWork struct {
	Work *Work `json:"work"`
}

func (*CreateWork) isAction() {}

type UpdateWork struct {
	Work *Work `json:"work"`
}

func (*UpdateWork) isAction() {}
