package backend

import (
	"context"
	"fmt"
	"strings"

	log "github.com/hashicorp/go-hclog"
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

	sender := FakeKafka{conf.Logger}

	b.Paths = framework.PathAppend(
		layerBackendPaths(b, tenantKeys, &TenantSchema{}, sender),
		layerBackendPaths(b, userKeys, &UserSchema{}, sender),
		layerBackendPaths(b, projectKeys, &ProjectSchema{}, sender),
		layerBackendPaths(b, serviceAccountKeys, &ServiceAccountSchema{}, sender),
		layerBackendPaths(b, groupKeys, &GroupSchema{}, sender),
		layerBackendPaths(b, roleKeys, &RoleSchema{}, sender),
	)

	return b
}

const commonHelp = `
IAM API here
`

type FakeKafka struct {
	logger log.Logger
}

func (f FakeKafka) Send(ctx context.Context, marshaller EntityMarshaller, topics []Topic) error {
	f.logger.Debug("sending to store", "key", marshaller.Key())
	return nil
}

func (f FakeKafka) Delete(ctx context.Context, marshaller EntityMarshaller, topics []Topic) error {
	f.logger.Debug("sending to delete", "key", marshaller.Key())
	return nil
}
