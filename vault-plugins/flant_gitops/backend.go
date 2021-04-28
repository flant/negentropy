package flant_gitops

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks"
)

type backend struct {
	*framework.Backend

	BackendCtx       context.Context
	BackendCtxCancel context.CancelFunc

	ConvergeTasks *tasks.VaultBackendTasks
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
	backendCtx, backendCtxCancel := context.WithCancel(context.Background())

	b := &backend{
		BackendCtx:       backendCtx,
		BackendCtxCancel: backendCtxCancel,
		ConvergeTasks:    tasks.NewVaultBackendTasks(backendCtx, nil),
	}

	b.Backend = &framework.Backend{
		PeriodicFunc: func(ctx context.Context, req *logical.Request) error {
			return Converge(b, ctx, req)
		},
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathConfigure(b),
			},
		),
	}

	return b, nil
}
