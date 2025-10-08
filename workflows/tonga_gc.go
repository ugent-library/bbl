package workflows

import (
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/ugent-library/tonga"
)

type TongaGCInput struct{}

type TongaGCOutput struct{}

func TongaGC(client *hatchet.Client, tongaClient *tonga.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("tonga_gc", func(ctx hatchet.Context, input TongaGCInput) (TongaGCOutput, error) {
		out := TongaGCOutput{}

		return out, tongaClient.GC(ctx)
	},
		hatchet.WithWorkflowCron("0 * * * *"),
		hatchet.WithWorkflowDescription("Hourly tonga queues cleanup"),
	)
}
