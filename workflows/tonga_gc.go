package workflows

import (
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/catbird"
)

type CatbirdGCInput struct{}

type CatbirdGCOutput struct{}

func CatbirdGC(client *hatchet.Client, catbirdClient *catbird.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("catbird_gc", func(ctx hatchet.Context, input CatbirdGCInput) (CatbirdGCOutput, error) {
		out := CatbirdGCOutput{}

		return out, catbirdClient.GC(ctx)
	},
		hatchet.WithWorkflowCron("0 * * * *"),
		hatchet.WithWorkflowDescription("Hourly catbird queues cleanup"),
	)
}
