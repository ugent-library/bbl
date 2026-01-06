package tasks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
)

const NotifySubscribersName = "notify_subscribers"

type NotifySubscribersInput struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

type NotifySubscribersOutput struct{}

// TODO work:changed only if public
func NotifySubscribers(repo *pgxrepo.Repo) *catbird.Task {
	return catbird.NewTask(NotifySubscribersName, func(ctx context.Context, input NotifySubscribersInput) (NotifySubscribersOutput, error) {
		out := NotifySubscribersOutput{}

		var err error

		for sub := range repo.TopicSubscriptionsIter(ctx, input.Topic, &err) {
			// TODO handle error
			repo.Catbird.RunTask(ctx, NotifySubscriberName, NotifySubscriberInput{
				WebhookURL: sub.WebhookURL,
				Topic:      input.Topic,
				Payload:    input.Payload,
			}, catbird.RunTaskOpts{}) // TODO set deduplication id
		}

		return out, err
	},
		catbird.TaskOpts{
			HideFor: 1 * time.Minute,
		},
	)
}
