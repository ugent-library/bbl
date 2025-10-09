package pgxrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/tonga"
)

type Conn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}

type Repo struct {
	conn  Conn
	Tonga *tonga.Client
}

func New(ctx context.Context, conn Conn) (*Repo, error) {
	tongaClient := tonga.New(conn)

	err := tongaClient.CreateQueue(ctx, bbl.OutboxQueue, []string{
		bbl.OrganizationChangedTopic,
		bbl.PersonChangedTopic,
		bbl.ProjectChangedTopic,
		bbl.WorkChangedTopic,
	}, tonga.QueueOpts{})
	if err != nil {
		return nil, err
	}

	return &Repo{
		conn:  conn,
		Tonga: tongaClient,
	}, nil
}
