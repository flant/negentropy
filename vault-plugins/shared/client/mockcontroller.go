package client

import (
	"context"

	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type MockVaultClientController struct {
	Client *api.Client
}

func (m *MockVaultClientController) GetApiConfig(context.Context) (*VaultApiConf, error) {
	panic("implement me") // nolint:panic_check
}

func (m *MockVaultClientController) APIClient() (*api.Client, error) {
	return m.Client, nil
}

func (m *MockVaultClientController) ReInit() error {
	panic("implement me") // nolint:panic_check
}

func (m *MockVaultClientController) UpdateOutdated(context.Context) error {
	panic("implement me") // nolint:panic_check
}

func (m *MockVaultClientController) HandleConfigureVaultAccess(context.Context, *logical.Request, *framework.FieldData) (*logical.Response, error) {
	panic("implement me") // nolint:panic_check
}
