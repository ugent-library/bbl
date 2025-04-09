package jobs

import (
	"context"
	"encoding/json"
	"time"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl"
	"github.com/ugent-library/tonga"
	"golang.org/x/sync/errgroup"
)

type ReindexPeopleArgs struct {
	// Strings is a slice of strings to sort.
	Strings []string `json:"strings"`
}

func (ReindexPeopleArgs) Kind() string { return "reindex_people" }

type ReindexPeopleWorker struct {
	river.WorkerDefaults[ReindexPeopleArgs]
	repo  *bbl.Repo
	index bbl.Index
}

func NewReindexPeopleWorker(repo *bbl.Repo, index bbl.Index) *ReindexPeopleWorker {
	return &ReindexPeopleWorker{
		repo:  repo,
		index: index,
	}
}

func (w *ReindexPeopleWorker) Work(ctx context.Context, job *river.Job[ReindexPeopleArgs]) error {
	channel := "people_reindexer"
	topic := "person"

	switcher, err := w.index.People().NewSwitcher(ctx)
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
			rec, err := w.repo.GetPerson(cancelCtx, id)
			if err != nil {
				break
			}
			if err = w.index.People().Add(cancelCtx, rec); err != nil {
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

		iter := w.repo.PeopleIter(groupCtx, &err)

		for rec := range iter {
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
