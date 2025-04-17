package jobs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/tonga"
	"golang.org/x/sync/errgroup"
)

type ReindexOrganizationsArgs struct{}

func (ReindexOrganizationsArgs) Kind() string { return "reindex_organizations" }

// only allow a new job when previous one completes
func (ReindexOrganizationsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRunning,
				rivertype.JobStateRetryable,
				rivertype.JobStateScheduled,
			},
		},
	}
}

type ReindexOrganizationsWorker struct {
	river.WorkerDefaults[ReindexOrganizationsArgs]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewReindexOrganizationsWorker(repo *pgxrepo.Repo, index bbl.Index) *ReindexOrganizationsWorker {
	return &ReindexOrganizationsWorker{
		repo:  repo,
		index: index,
	}
}

func (w *ReindexOrganizationsWorker) Work(ctx context.Context, job *river.Job[ReindexOrganizationsArgs]) error {
	channel := "organizations_reindexer"
	topic := "organization"

	switcher, err := w.index.Organizations().NewSwitcher(ctx)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	cancelCtx, cancel := context.WithCancel(groupCtx)

	group.Go(func() error {
		queue := w.repo.Queue()

		// TODO channel with ttl is more robust
		if err := queue.CreateChannel(cancelCtx, channel, topic, tonga.ChannelOpts{}); err != nil {
			return err
		}
		defer queue.DeleteChannel(cancelCtx, channel)

		var err error
		for msg := range w.repo.Listen(cancelCtx, channel, 10*time.Second) {
			var id string
			if err = json.Unmarshal(msg.Body, &id); err != nil {
				break
			}
			rec, err := w.repo.GetOrganization(cancelCtx, id)
			if err != nil {
				break
			}
			if err = w.index.Organizations().Add(cancelCtx, rec); err != nil {
				break
			}
			if _, err = w.repo.Queue().Delete(cancelCtx, channel, msg.ID); err != nil {
				break
			}
		}
		return err
	})

	group.Go(func() error {
		defer cancel()

		var err error

		for rec := range w.repo.OrganizationsIter(groupCtx, &err) {
			if err = switcher.Add(groupCtx, rec); err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		return switcher.Switch(groupCtx)
	})

	return group.Wait()
}
