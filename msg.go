package bbl

import (
	"encoding/json"
	"time"
)

type Msg struct {
	queue     string
	id        int64
	Topic     string
	Body      json.RawMessage
	CreatedAt time.Time
}
