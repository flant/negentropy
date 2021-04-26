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

	b := newBackend(conf)

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend(conf *logical.BackendConfig) logical.Backend {
	b := &framework.Backend{
		Help:        strings.TrimSpace(commonHelp),
		BackendType: logical.TypeLogical,
	}
	var (
		tenantKeys         = key.NewManager("tenant_id", "tenant")
		userKeys           = tenantKeys.Child("user_id", "user")
		projectKeys        = tenantKeys.Child("project_id", "project")
		serviceAccountKeys = tenantKeys.Child("service_account_id", "service_account")
		groupKeys          = tenantKeys.Child("group_id", "group")
		roleKeys           = tenantKeys.Child("role_id", "role")
	)


	b.Paths = framework.PathAppend(
		tenantPaths(b),
		layerBackendPaths(b, userKeys, &UserSchema{}),
		layerBackendPaths(b, projectKeys, &ProjectSchema{}),
		layerBackendPaths(b, serviceAccountKeys, &ServiceAccountSchema{}),
		layerBackendPaths(b, groupKeys, &GroupSchema{}),
		layerBackendPaths(b, roleKeys, &RoleSchema{}),
	)

	return b
}

const commonHelp = `
IAM API here
`
