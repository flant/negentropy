package flant_gitops

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/trdl/server/pkg/git"
	"github.com/werf/trdl/server/pkg/pgp"

	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/git_repository"
	"github.com/flant/negentropy/vault-plugins/flant_gitops/pkg/vault"
	"github.com/flant/negentropy/vault-plugins/shared/client"
)

type backend struct {
	*framework.Backend
	AccessVaultClientProvider client.AccessVaultClientController
	periodicTaskMutex         sync.Mutex
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend(conf)
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, fmt.Errorf("Configuration passed into backend is nil")
	}

	if err := b.SetupBackend(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend(c *logical.BackendConfig) (*backend, error) {
	accessVaultClientProvider, err := client.NewAccessVaultClientController(c.StorageView, hclog.Default())
	if err != nil {
		return nil, err
	}
	b := &backend{
		AccessVaultClientProvider: accessVaultClientProvider,
	}

	baseBackend := &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        backendHelp,

		PeriodicFunc: func(ctx context.Context, req *logical.Request) error {
			if err := b.AccessVaultClientProvider.UpdateOutdated(ctx); err != nil {
				return err
			}

			return b.PeriodicTask(req.Storage)
		},
	}

	baseBackend.Paths = framework.PathAppend(
		git_repository.ConfigurePaths(baseBackend),
		vault.ConfigurePaths(baseBackend),
		git.CredentialsPaths(),
		pgp.Paths(),
		[]*framework.Path{
			client.PathConfigure(b.AccessVaultClientProvider),
		},
	)

	b.Backend = baseBackend

	return b, nil
}

func (b *backend) SetupBackend(ctx context.Context, config *logical.BackendConfig) error {
	if err := b.Setup(ctx, config); err != nil {
		return err
	}

	return nil
}

const (
	backendHelp = `
The flant_gitops plugin starts an operator which waits for new commits in the configured git repository, verifies commit signatures by configured pgp keys, then executes configured commands in this new commit.
`
)
