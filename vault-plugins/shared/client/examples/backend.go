package jwt

import (
	"context"
	"errors"

	"github.com/flant/negentropy/vault-plugins/shared/client"
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
	accessVaultController *client.VaultClientController
}

func backend() *exampleBackend {
	b := new(exampleBackend)
	b.accessVaultController = client.NewVaultClientController(log.L)

	b.Backend = &framework.Backend{
		BackendType:  logical.TypeCredential,
		Help:         backendHelp,
		PathsSpecial: &logical.Paths{},

		PeriodicFunc: func(ctx context.Context, request *logical.Request) error {
			// MUST be called in periodical function
			// otherwise access token do not prolong
			return b.accessVaultController.OnPeriodical(ctx, request)
		},

		Paths: framework.PathAppend(
			[]*framework.Path{
				// NEED add /configure_vault_access path handler
				// for set configuration
				client.PathConfigure(b.accessVaultController),
			},

			[]*framework.Path{
				getPath(b),
				getConfPath(b),
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
				Summary:  "test getting role through api",
			},

			logical.UpdateOperation: &framework.PathOperation{
				Callback: b.pathReInit,
				Summary:  "test getting role through api after reinit client",
			},
		},

		HelpSynopsis:    "Syn",
		HelpDescription: "Desc",
	}
}

func getConfPath(b *exampleBackend) *framework.Path {
	return &framework.Path{
		Pattern: `get_conf$`,
		Fields:  map[string]*framework.FieldSchema{},

		Operations: map[logical.Operation]framework.OperationHandler{
			logical.ReadOperation: &framework.PathOperation{
				Callback: b.pathReadVaultApiConf,
				Summary:  "test getting vault api configuration",
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

	// Init access controller
	err = b.accessVaultController.Init(config.StorageView)
	// init may be return ErrNotSetConf error
	// if plugin initialized first time and has not saved config
	// its normal behavior. Because we set configuration
	// through "/configure_vault_access" path
	if err != nil && !errors.Is(err, client.ErrNotSetConf) {
		return err
	}

	return nil
}

func (b *exampleBackend) pathReadClientRole(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	apiClient, err := b.accessVaultController.APIClient()
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

func (b *exampleBackend) pathReadVaultApiConf(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	apiConf, err := b.accessVaultController.GetApiConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"info": "Getting vault api config",
			"res": map[string]interface{}{
				"hasConfig": apiConf != nil,
				"content":   apiConf,
			},
		},
	}, nil
}

// dont need in your backend use for test
func (b *exampleBackend) pathReInit(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	apiClient, err := b.accessVaultController.APIClient()
	if err != nil {
		return nil, err
	}

	err = apiClient.Auth().Token().RevokeSelf("" /* ignored */)
	if err != nil {
		return nil, err
	}

	err = b.accessVaultController.ReInit(req.Storage)
	if err != nil {
		return nil, err
	}

	apiClient, err = b.accessVaultController.APIClient()
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
