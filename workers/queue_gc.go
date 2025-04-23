package workers

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/bbl/jobs"
	"github.com/ugent-library/tonga"
)

type QueueGc struct {
	river.WorkerDefaults[jobs.QueueGc]
	queue *tonga.Client
}

func NewQueueGc(queue *tonga.Client) *QueueGc {
	return &QueueGc{
		queue: queue,
	}
}

func (w *QueueGc) Work(ctx context.Context, job *river.Job[jobs.QueueGc]) error {
	return w.queue.GC(ctx)
}
