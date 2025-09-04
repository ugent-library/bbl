package jobs

import (
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

type ImportWork struct {
	Source string `json:"source"`
	ID     string `json:"id"`
}

func (ImportWork) Kind() string { return "import_work" }

func (ImportWork) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
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

type ImportWorkOutput struct {
	WorkID string `json:"work_id"`
}
