package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const NotifySubscriberName = "notify_subscriber"

type NotifySubscriberInput struct {
	WebhookURL string          `json:"webhook_url"`
	Topic      string          `json:"topic"`
	Payload    json.RawMessage `json:"payload"`
}

type NotifySubscriberOutput struct{}

// TODO allow custom headers for auth etc
// TODO add signature header (  const hmac = crypto.createHmac('sha256', process.env.ZOOM_WEBHOOK_SECRET).update(ctx.rawBody).digest('hex'); )
func NotifySubscriber(repo *pgxrepo.Repo) *catbird.Task {
	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	return catbird.NewTask(NotifySubscriberName, func(ctx context.Context, input NotifySubscriberInput) (NotifySubscriberOutput, error) {
		out := NotifySubscriberOutput{}

		var err error

		b, err := json.Marshal(struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}{
			Topic:   input.Topic,
			Payload: input.Payload,
		})
		if err != nil {
			return out, err
		}

		req, err := http.NewRequest(http.MethodPost, input.WebhookURL, bytes.NewBuffer(b))
		if err != nil {
			return out, err
		}
		req.Header.Add("Content-Type", "application/json")
		_, err = httpClient.Do(req)
		if err != nil {
			return out, err
		}

		return out, err
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
			Retries: 2,
		},
	)
}
