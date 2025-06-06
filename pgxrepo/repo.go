package pgxrepo

import (
	"context"
	"iter"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/ugent-library/tonga"
)

type pgxConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

type Repo struct {
	conn        *pgxpool.Pool
	queue       *tonga.Client
	riverClient *river.Client[pgx.Tx]
}

func New(ctx context.Context, conn *pgxpool.Pool) (*Repo, error) {
	// insert only client
	// TODO pass logger, duplicates newInsertOnlyRiverClient
	riverClient, err := river.NewClient(riverpgxv5.New(conn), &river.Config{})
	if err != nil {
		return nil, err
	}

	return &Repo{
		conn:        conn,
		queue:       tonga.New(conn),
		riverClient: riverClient,
	}, nil
}

func (r *Repo) AddJob(ctx context.Context, job river.JobArgs) (int64, error) {
	res, err := r.riverClient.Insert(ctx, job, nil)
	if err != nil {
		return 0, err
	}
	return res.Job.ID, nil
}

func (r *Repo) Queue() *tonga.Client {
	return r.queue
}

// TODO tonga itself should have a higher level method
func (r *Repo) Listen(ctx context.Context, queue string, hideFor time.Duration) iter.Seq[*tonga.Message] {
	return func(yield func(*tonga.Message) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// TODO make quantity configurable
				msgs, err := r.queue.Read(ctx, queue, 10, hideFor)
				if err != nil {
					// TODO error handling
					log.Printf("listen: %s", err)
					return
				}

				for _, msg := range msgs {
					if ok := yield(msg); !ok {
						return
					}
				}

				if len(msgs) < 10 {
					// TODO make backoff configurable
					time.Sleep(1 * time.Second)
				}
			}
		}
	}
}
