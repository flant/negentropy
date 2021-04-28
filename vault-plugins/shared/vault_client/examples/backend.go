package jwt

import (
	"context"
	"errors"

	"github.com/flant/negentropy/vault-plugins/shared/vault_client"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

// Factory is used by framework
func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.SetupBackend(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

// Simple backend for test purposes (treat it like an example)
type exampleBackend struct {
	*framework.Backend
	accessVaultController *vault_client.VaultClientController
}

func backend() *exampleBackend {
	b := new(exampleBackend)
	b.accessVaultController = vault_client.NewVaultClientController(func() log.Logger {
		return b.Logger()
	})

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		Help:         backendHelp,
		PathsSpecial: &logical.Paths{},
		PeriodicFunc: func(ctx context.Context, request *logical.Request) error {
			return b.accessVaultController.OnPeriodical(ctx, request)
		},
		Paths: framework.PathAppend(
			[]*framework.Path{
				vault_client.PathConfigure(b.accessVaultController),
			},

			[]*framework.Path{
				getPath(b),
			},
		),
	}

	return b
}

func getPath(b *exampleBackend) *framework.Path {
	return &framework.Path{
		Pattern: `read_role$`,
		Fields:  map[string]*framework.FieldSchema{},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathReadClientRole,
				Summary:  "Read authentication source.",
			},
		},

		HelpSynopsis:    "Syn",
		HelpDescription: "Desc",
	}
}

func (b *exampleBackend) SetupBackend(ctx context.Context, config *logical.BackendConfig) error {
	err := b.Setup(ctx, config)
	if err != nil {
		return err
	}

	err = b.accessVaultController.Init(config.StorageView)
	if err != nil && !errors.Is(err, vault_client.NotSetConfError) {
		return err
	}

	return nil
}

func (b *exampleBackend) pathReadClientRole(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	apiClient, err := b.accessVaultController.ApiClient()
	if err != nil {
		return nil, err
	}

	res, err := apiClient.Logical().Read("/auth/approle/role/good/role-id")

	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"info": "Getting by client",
			"res":  res.Data,
			"client": map[string]interface{}{
				"address": apiClient.Address(),
				"headers": apiClient.Headers(),
			},
		},
	}, nil
}

const (
	backendHelp = `
Example clientApi backend
`
)
