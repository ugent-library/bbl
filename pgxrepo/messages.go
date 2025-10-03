package pgxrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/ugent-library/bbl"
)

func (r *Repo) ReadMessages(ctx context.Context, n int, hideFor time.Duration) ([]bbl.Message, error) {
	q := `
		WITH msgs AS
        (
            SELECT id
            FROM bbl_messages
            WHERE deliver_at <= clock_timestamp()
            ORDER BY id ASC
            LIMIT $1
            FOR UPDATE SKIP LOCKED
        )
        UPDATE bbl_messages m
        SET
            deliver_at = clock_timestamp() + make_interval(secs => $2)
        FROM msgs
        WHERE m.id = msgs.id
        RETURNING m.id, m.topic, m.payload, m.created_at;
	`

	rows, err := r.conn.Query(ctx, q, n, hideFor.Seconds())
	if err != nil {
		return nil, fmt.Errorf("ReadMessages: %s", err)
	}

	msgs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[bbl.Message])
	if err != nil {
		return nil, fmt.Errorf("ReadMessages: %s", err)
	}

	return msgs, nil
}

func (r *Repo) DeleteMessage(ctx context.Context, id int64) error {
	q := `DELETE FROM bbl_messages WHERE id = $1;`

	if _, err := r.conn.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("DeleteMessage: %s", err)
	}

	return nil
}

func enqueueMessage(batch *pgx.Batch, topic string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	batch.Queue(
		`INSERT INTO bbl_messages (topic, payload) VALUES ($1, $2);`,
		topic, b,
	)

	return nil
}
