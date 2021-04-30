package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks"
)

type backend struct {
	*framework.Backend
	TaskQueueBackend *tasks.Backend
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
		TaskQueueBackend: tasks.NewBackend(),
	}

	b.Backend = &framework.Backend{
		PeriodicFunc: b.TaskQueueBackend.PeriodicFunc(b.periodicTask),
		BackendType:  logical.TypeLogical,
		Paths: framework.PathAppend(
			configurePaths(b),
			b.TaskQueueBackend.Paths(),
		),
	}

	return b, nil
}
