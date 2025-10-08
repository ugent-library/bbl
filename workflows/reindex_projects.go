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

type ReindexProjectsInput struct {
}

type ReindexProjectsOutput struct {
}

func ReindexProjects(client *hatchet.Client, repo *pgxrepo.Repo, index bbl.Index) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("reindex_projects", func(ctx hatchet.Context, input ReindexProjectsInput) (ReindexProjectsOutput, error) {
		out := ReindexProjectsOutput{}
		queue := "projects_reindexer_" + time.Now().UTC().Format(timeFormat)
		topic := bbl.ProjectChangedTopic

		switcher, err := index.Projects().NewSwitcher(ctx)
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

						rec, err := repo.GetProject(groupCtx, payload.ID)
						if err != nil {
							return err
						}
						if err = index.Projects().Add(groupCtx, rec); err != nil {
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

			for rec := range repo.ProjectsIter(groupCtx, &err) {
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
