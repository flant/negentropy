package flant_gitops

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"

	"github.com/flant/negentropy/vault-plugins/shared/client"
)

type backend struct {
	*framework.Backend
	TasksManager          tasks_manager.Interface
	AccessVaultController *client.VaultClientController

	LastPeriodicTaskUUID string
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	if err := b.SetupBackend(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{
		TasksManager:          tasks_manager.NewManager(),
		AccessVaultController: client.NewVaultClientController(hclog.L),
	}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        "TODO",

		PeriodicFunc: func(ctx context.Context, req *logical.Request) error {
			if err := b.AccessVaultController.OnPeriodical(ctx, req); err != nil {
				return err
			}

			if err := b.TasksManager.PeriodicTask(ctx, req); err != nil {
				return err
			}

			if err := b.PeriodicTask(req); err != nil {
				return err
			}

			return nil
		},

		Paths: framework.PathAppend(
			configurePaths(b),
			configureGitCredentialPaths(b),
			configureVaultRequestPaths(b),
			b.TasksManager.Paths(),
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
