package bbl

import "time"

type List struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Public      bool      `json:"public"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedByID string    `json:"created_by_id"`
}
