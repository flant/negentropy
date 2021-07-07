package flant_gitops

import (
	"context"
	"errors"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager"

	"github.com/flant/negentropy/vault-plugins/shared/client"
)

type backend struct {
	*framework.Backend
	TaskQueueManager      queue_manager.Interface
	AccessVaultController *client.VaultClientController

	LastPeriodicTaskUUID string
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if err := b.SetupBackend(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{
		TaskQueueManager:      queue_manager.NewManager(),
		AccessVaultController: client.NewVaultClientController(hclog.L),
	}

	b.Backend = &framework.Backend{
		PeriodicFunc: func(ctx context.Context, req *logical.Request) error {
			if err := b.AccessVaultController.OnPeriodical(ctx, req); err != nil {
				return err
			}

			if err := b.PeriodicTask(req); err != nil {
				return err
			}

			return nil
		},
		BackendType: logical.TypeLogical,
		Paths: framework.PathAppend(
			configurePaths(b),
			b.TaskQueueManager.Paths(),
			pgp.Paths(),
			[]*framework.Path{
				client.PathConfigure(b.AccessVaultController),
			},
		),
	}

	return b, nil
}

func (b *backend) SetupBackend(ctx context.Context, config *logical.BackendConfig) error {
	if err := b.Setup(ctx, config); err != nil {
		return err
	}

	if err := b.AccessVaultController.Init(config.StorageView); err != nil && !errors.Is(err, client.ErrNotSetConf) {
		return err
	}

	return nil
}
