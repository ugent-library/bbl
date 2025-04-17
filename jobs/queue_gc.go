package jobs

import (
	"context"

	"github.com/riverqueue/river"
	"github.com/ugent-library/tonga"
)

type QueueGcArgs struct{}

func (QueueGcArgs) Kind() string { return "queue_gc" }

type QueueGcWorker struct {
	river.WorkerDefaults[QueueGcArgs]
	queue *tonga.Client
}

func NewQueueGcWorker(queue *tonga.Client) *QueueGcWorker {
	return &QueueGcWorker{
		queue: queue,
	}
}

func (w *QueueGcWorker) Work(ctx context.Context, job *river.Job[QueueGcArgs]) error {
	return w.queue.GC(ctx)
}
