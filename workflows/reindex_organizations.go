package workflows

import (
	"context"
	"encoding/json"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/tonga"
	"golang.org/x/sync/errgroup"
)

type ReindexOrganizationsInput struct {
}

type ReindexOrganizationsOutput struct {
}

func ReindexOrganizations(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("reindex_organizations", func(ctx hatchet.Context, input ReindexOrganizationsInput) (ReindexOrganizationsOutput, error) {
		out := ReindexOrganizationsOutput{}
		queue := "organizations_reindexer_" + time.Now().UTC().Format(timeFormat)
		topic := bbl.OrganizationChangedTopic

		switcher, err := index.Organizations().NewSwitcher(ctx)
		if err != nil {
			return out, err
		}

		group, groupCtx := errgroup.WithContext(ctx)
		cancelCtx, cancel := context.WithCancel(groupCtx)

		group.Go(func() error {
			n := 100
			hideFor := 10 * time.Second

			queueOpts := tonga.QueueOpts{
				DeleteAt: time.Now().Add(30 * time.Minute),
				Unlogged: true,
			}
			if err := repo.Tonga.CreateQueue(groupCtx, queue, []string{topic}, queueOpts); err != nil {
				return err
			}

			for {
				select {
				case <-cancelCtx.Done():
					return nil
				default:
					msgs, err := repo.Tonga.Read(groupCtx, queue, n, hideFor)
					if err != nil {
						return err
					}

					for _, msg := range msgs {
						var payload bbl.RecordChangedPayload
						if err = json.Unmarshal(msg.Payload, &payload); err != nil {
							return err
						}

						rec, err := repo.GetOrganization(groupCtx, payload.ID)
						if err != nil {
							return err
						}
						if err = index.Organizations().Add(groupCtx, rec); err != nil {
							return err
						}

						if _, err = repo.Tonga.Delete(groupCtx, queue, msg.ID); err != nil {
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

			for rec := range repo.OrganizationsIter(groupCtx, &err) {
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
