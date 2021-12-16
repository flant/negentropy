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

func (m *MockVaultClientController) GetApiConfig(ctx context.Context, storage logical.Storage) (*VaultApiConf, error) {
	panic("implement me")
}

func (m *MockVaultClientController) APIClient(storage logical.Storage) (*api.Client, error) {
	return m.Client, nil
}

func (m *MockVaultClientController) ReInit(storage logical.Storage) error {
	panic("implement me")
}

func (m *MockVaultClientController) OnPeriodical(ctx context.Context, request *logical.Request) error {
	panic("implement me")
}

func (m *MockVaultClientController) HandleConfigureVaultAccess(ctx context.Context, request *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	panic("implement me")
}
