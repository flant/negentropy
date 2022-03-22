package api

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/hashicorp/vault/sdk/physical/inmem"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend"
	url2 "github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/url"
	"github.com/flant/negentropy/vault-plugins/shared/tests"
)

func NewRoleAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.RoleEndpointBuilder{}}
}

func NewFeatureFlagAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.FeatureFlagEndpointBuilder{}}
}

func NewTenantAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.TenantEndpointBuilder{}}
}

func NewIdentitySharingAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.IdentitySharingEndpointBuilder{}}
}

func NewRoleBindingAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.RoleBindingEndpointBuilder{}}
}

func NewRoleBindingApprovalAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.RoleBindingApprovalEndpointBuilder{}}
}

func NewTenantFeatureFlagAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.TenantFeatureFlagEndpointBuilder{}}
}

func NewUserAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.UserEndpointBuilder{}}
}

func NewGroupAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.GroupEndpointBuilder{}}
}

func NewUserMultipassAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.UserMultipassEndpointBuilder{}}
}

func NewProjectAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ProjectEndpointBuilder{}}
}

func NewProjectFeatureFlagAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ProjectFeatureFlagEndpointBuilder{}}
}

func NewServerAPI(b *logical.Backend, s *logical.Storage) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ServerEndpointBuilder{}, Storage: s}
}

func NewServiceAccountAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ServiceAccountEndpointBuilder{}}
}

func NewServiceAccountPasswordAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ServiceAccountPasswordEndpointBuilder{}}
}

func NewServiceAccountMultipassAPI(b *logical.Backend) tests.TestAPI {
	return &tests.BackendBasedAPI{Backend: b, Url: &url2.ServiceAccountMultipassEndpointBuilder{}}
}

type VaultPayload struct {
	Data json.RawMessage `json:"data"`
}

func TestBackend() logical.Backend {
	config := logical.TestBackendConfig()
	testPhisicalBackend, _ := inmem.NewInmemHA(map[string]string{}, config.Logger)
	config.StorageView = logical.NewStorageView(logical.NewLogicalStorage(testPhisicalBackend), "")
	b, err := backend.Factory(context.Background(), config)
	if err != nil {
		panic(err)
	}
	return b
}

func TestBackendWithStorage() (logical.Backend, logical.Storage) {
	config := logical.TestBackendConfig()
	testPhisicalBackend, _ := inmem.NewInmemHA(map[string]string{}, config.Logger)
	config.StorageView = logical.NewStorageView(logical.NewLogicalStorage(testPhisicalBackend), "")
	b, err := backend.Factory(context.Background(), config)
	if err != nil {
		panic(err)
	}
	return b, config.StorageView
}
