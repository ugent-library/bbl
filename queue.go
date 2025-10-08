package bbl

const (
	OutboxQueue = "outbox"

	OrganizationChangedTopic = "organization:changed"
	PersonChangedTopic       = "person:changed"
	ProjectChangedTopic      = "project:changed"
	WorkChangedTopic         = "work:changed"
)

type RecordChangedPayload struct {
	Rev string `json:"rev"`
	ID  string `json:"id"`
}
