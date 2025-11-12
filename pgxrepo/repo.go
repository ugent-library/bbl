package pgxrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/catbird"
)

type Conn interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

type Repo struct {
	conn    Conn
	Catbird *catbird.Client
}

func New(ctx context.Context, conn Conn) (*Repo, error) {
	catbirdClient := catbird.New(conn)

	err := catbirdClient.CreateQueue(ctx, bbl.OutboxQueue, []string{
		bbl.OrganizationChangedTopic,
		bbl.PersonChangedTopic,
		bbl.ProjectChangedTopic,
		bbl.WorkChangedTopic,
	}, catbird.QueueOpts{})
	if err != nil {
		return nil, err
	}

	return &Repo{
		conn:    conn,
		Catbird: catbirdClient,
	}, nil
}
