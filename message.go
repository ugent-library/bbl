package bbl

import (
	"encoding/json"
	"time"
)

type Message struct {
	ID        int64           `json:"id"`
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

const (
	OrganizationChangedTopic = "organization:changed"
	PersonChangedTopic       = "person:changed"
	ProjectChangedTopic      = "project:changed"
	WorkChangedTopic         = "work:changed"
)

type RecordChangedPayload struct {
	Rev string `json:"rev"`
	ID  string `json:"id"`
}
