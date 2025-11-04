package workflows

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
)

type NotifySubscribersInput struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type NotifySubscribersOutput struct {
}

// TODO work:changed only if public
// TODO allow custom headers for auth etc
// TODO add signature header (  const hmac = crypto.createHmac('sha256', process.env.ZOOM_WEBHOOK_SECRET).update(ctx.rawBody).digest('hex'); )
// TODO send each notification only once with subtasks + dedup
func NotifySubscribers(client *hatchet.Client, repo *pgxrepo.Repo) *hatchet.StandaloneTask {
	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	return client.NewStandaloneTask("notify_subscribers", func(ctx hatchet.Context, input NotifySubscribersInput) (NotifySubscribersOutput, error) {
		out := NotifySubscribersOutput{}

		var err error

		b, err := json.Marshal(input)
		if err != nil {
			return out, err
		}

		for sub := range repo.TopicSubscriptionsIter(ctx, input.Topic, &err) {
			req, err := http.NewRequest(http.MethodPost, sub.WebhookURL, bytes.NewBuffer(b))
			if err != nil {
				return out, err
			}
			req.Header.Add("Content-Type", "application/json")
			_, err = httpClient.Do(req)
			if err != nil {
				return out, err
			}
		}

		return out, err
	},
		hatchet.WithWorkflowEvents(bbl.WorkChangedTopic),
	)
}
