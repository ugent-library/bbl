package bbl

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.breu.io/ulid"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrNotUnique = errors.New("not unique")
	ErrConflict  = errors.New("version conflict")
)

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
	Identifiers []Code    `json:"identifiers,omitempty"`
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
		case "save_user":
			action = &SaveUser{}
		case "save_organization":
			action = &SaveOrganization{}
		case "save_person":
			action = &SavePerson{}
		case "save_project":
			action = &SaveProject{}
		case "save_work":
			action = &SaveWork{}
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

type SaveUser struct {
	User         *User `json:"user"`
	MatchVersion bool  `json:"match_version"`
}

func (*SaveUser) isAction() {}

type SaveOrganization struct {
	Organization *Organization `json:"organization"`
	MatchVersion bool          `json:"match_version"`
}

func (*SaveOrganization) isAction() {}

type SavePerson struct {
	Person       *Person `json:"person"`
	MatchVersion bool    `json:"match_version"`
}

func (*SavePerson) isAction() {}

type SaveProject struct {
	Project      *Project `json:"project"`
	MatchVersion bool     `json:"match_version"`
}

func (*SaveProject) isAction() {}

type SaveWork struct {
	Work         *Work `json:"work"`
	MatchVersion bool  `json:"match_version"`
}

func (*SaveWork) isAction() {}

type ChangeWork struct {
	WorkID  string        `json:"work_id"`
	Version int           `json:"version,omitempty"`
	Changes []WorkChanger `json:"changes"`
}

func (*ChangeWork) isAction() {}

// TODO also use raw format here?
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
