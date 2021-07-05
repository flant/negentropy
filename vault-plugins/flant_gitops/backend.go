package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"
)

type backend struct {
	*framework.Backend
	TaskQueueManager queue_manager.Interface
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{
		TaskQueueManager: queue_manager.NewManager(),
	}

	b.Backend = &framework.Backend{
		PeriodicFunc: func(_ context.Context, req *logical.Request) error {
			uuid, err := b.TaskQueueManager.RunTask(context.Background(), req.Storage, b.periodicTask)
			if err != queue_manager.QueueBusyError {
				return err
			}

			_ = uuid
			_ = err

			return nil
		},
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			configurePaths(b),
			b.TaskQueueManager.Paths(),
		),
	}

	return b, nil
}
