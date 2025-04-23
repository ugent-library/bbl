package workers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/bbl/pgxrepo"
	"github.com/ugent-library/tonga"
	"golang.org/x/sync/errgroup"
)

type ReindexProjects struct {
	river.WorkerDefaults[jobs.ReindexProjects]
	repo  *pgxrepo.Repo
	index bbl.Index
}

func NewReindexProjects(repo *pgxrepo.Repo, index bbl.Index) *ReindexProjects {
	return &ReindexProjects{
		repo:  repo,
		index: index,
	}
}

func (w *ReindexProjects) Work(ctx context.Context, job *river.Job[jobs.ReindexProjects]) error {
	channel := "projects_reindexer"
	topic := "project"

	switcher, err := w.index.Projects().NewSwitcher(ctx)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)
	cancelCtx, cancel := context.WithCancel(groupCtx)

	group.Go(func() error {
		queue := w.repo.Queue()

		channelOpts := tonga.ChannelOpts{
			DeleteAt: time.Now().Add(30 * time.Minute),
			Unlogged: true,
		}
		if err := queue.CreateChannel(cancelCtx, channel, topic, channelOpts); err != nil {
			return err
		}
		defer queue.DeleteChannel(cancelCtx, channel)

		var err error
		for msg := range w.repo.Listen(cancelCtx, channel, 10*time.Second) {
			var id string
			if err = json.Unmarshal(msg.Body, &id); err != nil {
				break
			}
			rec, err := w.repo.GetProject(cancelCtx, id)
			if err != nil {
				break
			}
			if err = w.index.Projects().Add(cancelCtx, rec); err != nil {
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

		for rec := range w.repo.ProjectsIter(groupCtx, &err) {
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
