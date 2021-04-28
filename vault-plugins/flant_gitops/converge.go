package flant_gitops

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/logboek"
)

func Converge(backend *backend, ctx context.Context, req *logical.Request) error {
	backend.ConvergeTasks.Storage = req.Storage // This is actually the hack, that vault permits, that we agreed to use if not found a better way to access storage in the background tasks

	taskID := backend.ConvergeTasks.RunScheduledTask(func(ctx context.Context) error {
		logboek.Context(ctx).Default().LogF("Converge started\n")

		for i := 0; i < 25; i++ {
			time.Sleep(1 * time.Second)
			logboek.Context(ctx).Default().LogF("Converge in progress %d seconds\n", i+1)
		}

		logboek.Context(ctx).Default().LogF("Converge finished\n")

		return nil
	})

	if taskID != "" {
		// LOG: scheduled new converge task id=taskID
	}

	return nil
}
