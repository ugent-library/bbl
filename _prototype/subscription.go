package bbl

type Subscription struct {
	ID         string `json:"id,omitempty"`
	UserID     string `json:"user_id"`
	Topic      string `json:"topic"`
	WebhookURL string `json:"webhook_url"`
}
