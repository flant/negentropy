package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
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

	backendLayer := &layerBackend{b}

	tenantKeys := &keyManager{
		idField:   "tenant_id",
		entryName: "tenant",
	}

	userKeys := &keyManager{
		idField:   "user_id",
		entryName: "user",
		parent:    tenantKeys,
	}

	b.Paths = framework.PathAppend(
		backendLayer.paths(tenantKeys, TenantSchema{}),
		backendLayer.paths(userKeys, UserSchema{}),
	)

	return b
}

const commonHelp = `
IAM API here
`
