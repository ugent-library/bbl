package workflows

import (
	"context"
	"encoding/json"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/catbird"
	"golang.org/x/sync/errgroup"
)

type ReindexUsersInput struct{}

type ReindexUsersOutput struct{}

func ReindexUsers(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("reindex_users", func(ctx hatchet.Context, input ReindexUsersInput) (ReindexUsersOutput, error) {
		out := ReindexUsersOutput{}
		queue := "users_reindexer_" + time.Now().UTC().Format(timeFormat)
		topic := bbl.UserChangedTopic

		switcher, err := index.Users().NewSwitcher(ctx)
		if err != nil {
			return out, err
		}

		group, groupCtx := errgroup.WithContext(ctx)
		cancelCtx, cancel := context.WithCancel(groupCtx)

		group.Go(func() error {
			n := 100
			hideFor := 10 * time.Second

			queueOpts := catbird.QueueOpts{
				DeleteAt: time.Now().Add(30 * time.Minute),
				Unlogged: true,
			}
			if err := repo.Catbird.CreateQueue(groupCtx, queue, []string{topic}, queueOpts); err != nil {
				return err
			}

			for {
				select {
				case <-cancelCtx.Done():
					return nil
				default:
					msgs, err := repo.Catbird.Read(groupCtx, queue, n, hideFor)
					if err != nil {
						return err
					}

					for _, msg := range msgs {
						var payload bbl.RecordChangedPayload
						if err = json.Unmarshal(msg.Payload, &payload); err != nil {
							return err
						}

						rec, err := repo.GetUser(groupCtx, payload.ID)
						if err != nil {
							return err
						}
						if err = index.Users().Add(groupCtx, rec); err != nil {
							return err
						}

						if _, err = repo.Catbird.Delete(groupCtx, queue, msg.ID); err != nil {
							return err
						}
					}

					if len(msgs) < n {
						time.Sleep(500 * time.Millisecond)
					}
				}
			}
		})

		group.Go(func() error {
			defer cancel()

			var err error
			for rec := range repo.UsersIter(groupCtx, &err) {
				if err = switcher.Add(groupCtx, rec); err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}

			return switcher.Switch(groupCtx)
		})

		return out, group.Wait()
	},
	// hatchet.WithWorkflowConcurrency(types.Concurrency{
	// 	LimitStrategy: &strategyCancelNewest,
	// }),
	)
}
