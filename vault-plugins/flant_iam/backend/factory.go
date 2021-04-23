package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/key"
)

var _ logical.Factory = Factory

// Factory configures and returns Mock backends
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	b := newBackend()

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() logical.Backend {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}

	tenantKeys := key.NewManager("tenant_id", "tenant")

	b.Paths = framework.PathAppend(
		layerBackendPaths(b, tenantKeys, &TenantSchema{}),
		layerBackendPaths(b, tenantKeys.Child("user_id", "user"), &UserSchema{}),
		layerBackendPaths(b, tenantKeys.Child("project_id", "project"), &ProjectSchema{}),
		layerBackendPaths(b, tenantKeys.Child("service_account_id", "service_account"), &ServiceAccountSchema{}),
		layerBackendPaths(b, tenantKeys.Child("group_id", "group"), &GroupSchema{}),
		layerBackendPaths(b, tenantKeys.Child("role_id", "role"), &RoleSchema{}),
	)

	return b
}

func layerBackendPaths(b *framework.Backend, keyman *key.Manager, schema Schema) []*framework.Path {
	bb := &layerBackend{
		Backend: b,
		keyman:  keyman,
		schema:  schema,
	}
	return bb.paths()
}

const commonHelp = `
IAM API here
`
