package backend

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-memdb"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
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

	// Storage
	storage, err := memdb.NewMemDB(model.TenantSchema())
	if err != nil {
		panic(err)
	}

	// Paths
	b.Paths = framework.PathAppend(
		tenantPaths(b, storage),
	)

	return b
}

const commonHelp = `
IAM API here
`
