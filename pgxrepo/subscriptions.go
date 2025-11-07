package pgxrepo

import (
	"context"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) TopicSubscriptionsIter(ctx context.Context, topic string, errPtr *error) iter.Seq[*bbl.Subscription] {
	q := `SELECT ` + subscriptionCols + ` FROM bbl_subscriptions s WHERE s.topic = $1;`

	return func(yield func(*bbl.Subscription) bool) {
		rows, err := r.conn.Query(ctx, q, topic)
		if err != nil {
			*errPtr = err
			return
		}
		defer rows.Close()

		for rows.Next() {
			rec, err := scanSubscription(rows)
			if err != nil {
				*errPtr = err
				return
			}
			if !yield(rec) {
				return
			}
		}
	}
}

const subscriptionCols = `
	s.id,
	s.user_id,
	s.topic,
	s.webhook_url
`

func scanSubscription(row pgx.Row) (*bbl.Subscription, error) {
	var rec bbl.Subscription

	if err := row.Scan(&rec.ID, &rec.UserID, &rec.Topic, &rec.WebhookURL); err != nil {
		return nil, err
	}

	return &rec, nil
}
