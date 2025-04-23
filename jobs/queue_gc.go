package jobs

type QueueGc struct{}

func (QueueGc) Kind() string { return "queue_gc" }
